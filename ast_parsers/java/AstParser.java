import com.sun.source.tree.CompilationUnitTree;
import com.sun.source.tree.IdentifierTree;
import com.sun.source.tree.ImportTree;
import com.sun.source.tree.MemberSelectTree;
import com.sun.source.tree.Tree;
import com.sun.source.util.JavacTask;

import javax.tools.Diagnostic;
import javax.tools.DiagnosticCollector;
import javax.tools.JavaCompiler;
import javax.tools.JavaFileObject;
import javax.tools.StandardJavaFileManager;
import javax.tools.ToolProvider;
import java.io.FileDescriptor;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.PrintStream;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.LinkedHashSet;
import java.util.LinkedList;
import java.util.List;
import java.util.Set;
import java.util.stream.Stream;

/**
 * ArchLens parser for Java source files.
 *
 * <pre>java -jar javaparser.jar &lt;file&gt; [-r DIR | --root DIR]</pre>
 *
 * The file is parsed with javac's own parser through the Compiler Tree API
 * (com.sun.source), so imports come from a real AST and syntax errors are
 * javac's. Only parsing is done (no attribution), so no classpath is needed.
 *
 * Imports are resolved the way a source path lookup would, first internal hit wins:
 *  1. the source root implied by the file's own package declaration
 *     (file directory minus the package path),
 *  2. for the file's directory and every ancestor up to the project root:
 *     the directory itself plus its src/main/java, src/test/java and src.
 *
 *   import a.b.C;          -> a/b/C.java, else enclosing type (a.b.C.D -> a/b/C.java)
 *   import a.b.*;          -> directory a/b holding .java files, else the type a.b
 *   import static a.b.C.m; -> type a.b.C
 *   import static a.b.C.*; -> type a.b.C
 *
 * Directories named node_modules, vendor, .m2, .gradle, site-packages or .git
 * inside the root are third-party, and so is anything reached through a symlink
 * that leaves the root.
 */
public final class AstParser {

    static final class ParserError extends Exception {
        ParserError(String message) {
            super(message);
        }
    }

    static final Set<String> THIRD_PARTY =
            Set.of("node_modules", "vendor", ".m2", ".gradle", "site-packages", ".git");

    public static void main(String[] args) {
        String out;
        try {
            out = run(args);
        } catch (ParserError e) {
            System.err.println("javaparser: " + e.getMessage());
            System.exit(1);
            return;
        } catch (Exception e) {
            System.err.println("javaparser: " + e);
            System.exit(1);
            return;
        }
        PrintStream stdout = new PrintStream(new FileOutputStream(FileDescriptor.out), false, StandardCharsets.UTF_8);
        stdout.print(out);
        stdout.flush();
    }

    // ------------------------------------------------------------------ run

    static String run(String[] args) throws ParserError {
        String fileArg = null;
        String rootArg = ".";
        for (int i = 0; i < args.length; i++) {
            String a = args[i];
            if (a.equals("-r") || a.equals("--root")) {
                if (i + 1 >= args.length) throw new ParserError(a + " requires a directory");
                rootArg = args[++i];
            } else if (a.startsWith("--root=")) {
                rootArg = a.substring("--root=".length());
            } else if (a.startsWith("-r=")) {
                rootArg = a.substring("-r=".length());
            } else if (a.startsWith("-") && a.length() > 1) {
                throw new ParserError("unknown option '" + a + "'");
            } else if (fileArg != null) {
                throw new ParserError("expected exactly one file");
            } else {
                fileArg = a;
            }
        }
        if (fileArg == null) throw new ParserError("usage: javaparser <file> [-r DIR | --root DIR]");
        if (rootArg.isEmpty()) throw new ParserError("root must not be empty");

        // Root: absolute, symlink-free, a directory.
        Path root;
        try {
            root = Path.of(rootArg).toAbsolutePath().toRealPath();
        } catch (IOException | RuntimeException e) {
            throw new ParserError("root '" + rootArg + "' is not usable: " + e.getMessage());
        }
        if (!Files.isDirectory(root)) throw new ParserError("root '" + rootArg + "' is not a directory");

        // File: resolved against the working directory, symlink-free, inside the root.
        Path file;
        try {
            file = Path.of(fileArg).toAbsolutePath().toRealPath();
        } catch (IOException | RuntimeException e) {
            throw new ParserError("cannot read '" + fileArg + "': " + e.getMessage());
        }
        if (!Files.isRegularFile(file) || !Files.isReadable(file)) {
            throw new ParserError("'" + fileArg + "' is not a readable file");
        }
        Path fileDir = file.getParent();
        if (!fileDir.startsWith(root)) {
            throw new ParserError("file '" + fileArg + "' is outside the root '" + rootArg + "'");
        }
        if (!file.getFileName().toString().endsWith(".java")) {
            throw new ParserError("'" + fileArg + "' is not a .java file");
        }

        ParsedFile parsed = parse(file);

        Resolver resolver = new Resolver(root, searchRoots(root, fileDir, parsed.packageSegments));
        String own = packageName(root, fileDir);
        Set<String> deps = new LinkedHashSet<>();
        for (Import imp : parsed.imports) {
            String pkg = null;
            if (imp.isStatic) {
                List<String> segs = imp.segments;
                if (!imp.star) {
                    if (segs.size() < 2) continue;
                    segs = segs.subList(0, segs.size() - 1); // drop member name
                }
                pkg = resolver.resolveType(segs);
            } else if (imp.star) {
                pkg = resolver.resolvePackage(imp.segments);
                if (pkg == null) pkg = resolver.resolveType(imp.segments);
            } else {
                pkg = resolver.resolveType(imp.segments);
            }
            if (pkg != null && !pkg.equals(own)) deps.add(pkg);
        }
        return toJson(own, deps);
    }

    // ------------------------------------------------------------- parsing

    record Import(boolean isStatic, boolean star, List<String> segments) {}

    record ParsedFile(List<String> packageSegments, List<Import> imports) {}

    /** Parses with javac and extracts the package and import declarations from the AST. */
    static ParsedFile parse(Path file) throws ParserError {
        JavaCompiler compiler = ToolProvider.getSystemJavaCompiler();
        if (compiler == null) throw new ParserError("no system Java compiler found; run on a JDK, not a JRE");

        DiagnosticCollector<JavaFileObject> diagnostics = new DiagnosticCollector<>();
        try (StandardJavaFileManager fm =
                     compiler.getStandardFileManager(diagnostics, null, StandardCharsets.UTF_8)) {
            Iterable<? extends JavaFileObject> units = fm.getJavaFileObjects(file);
            JavacTask task = (JavacTask) compiler.getTask(
                    null, fm, diagnostics, List.of("-proc:none", "-encoding", "UTF-8"), null, units);

            List<CompilationUnitTree> trees = new ArrayList<>();
            for (CompilationUnitTree t : task.parse()) trees.add(t);

            for (Diagnostic<? extends JavaFileObject> d : diagnostics.getDiagnostics()) {
                if (d.getKind() == Diagnostic.Kind.ERROR) {
                    throw new ParserError(file + ":" + d.getLineNumber() + ": syntax error: " + d.getMessage(null));
                }
            }
            if (trees.size() != 1) throw new ParserError("could not parse " + file);

            CompilationUnitTree cu = trees.get(0);
            List<String> pkg = cu.getPackageName() == null ? List.of() : segments(cu.getPackageName());
            List<Import> imports = new ArrayList<>();
            for (ImportTree it : cu.getImports()) {
                List<String> segs = segments(it.getQualifiedIdentifier());
                boolean star = !segs.isEmpty() && segs.get(segs.size() - 1).equals("*");
                if (star) segs = segs.subList(0, segs.size() - 1);
                if (!segs.isEmpty()) imports.add(new Import(it.isStatic(), star, segs));
            }
            return new ParsedFile(pkg, imports);
        } catch (IOException e) {
            throw new ParserError("cannot read '" + file + "': " + e.getMessage());
        }
    }

    /** a.b.C (a MemberSelectTree chain ending in an IdentifierTree) -> [a, b, C]. */
    static List<String> segments(Tree tree) {
        LinkedList<String> out = new LinkedList<>();
        Tree t = tree;
        while (t instanceof MemberSelectTree ms) {
            out.addFirst(ms.getIdentifier().toString());
            t = ms.getExpression();
        }
        if (t instanceof IdentifierTree id) out.addFirst(id.getName().toString());
        return out;
    }

    // ------------------------------------------------------------ resolving

    static String packageName(Path root, Path dir) {
        Path rel = root.relativize(dir);
        if (rel.toString().isEmpty()) return ".";
        List<String> parts = new ArrayList<>();
        for (Path p : rel) parts.add(p.toString());
        return String.join("/", parts);
    }

    static List<Path> searchRoots(Path root, Path fileDir, List<String> pkg) {
        Set<Path> out = new LinkedHashSet<>();
        // 1. source root implied by the package declaration
        if (!pkg.isEmpty()) {
            Path dir = fileDir;
            boolean ok = true;
            for (int k = pkg.size() - 1; k >= 0 && ok; k--) {
                if (dir == null || dir.getFileName() == null || !dir.getFileName().toString().equals(pkg.get(k))) {
                    ok = false;
                } else {
                    dir = dir.getParent();
                }
            }
            if (ok && dir != null && dir.startsWith(root)) out.add(dir);
        }
        // 2. the file's directory and every ancestor up to the root
        Path dir = fileDir;
        while (dir != null) {
            out.add(dir);
            out.add(dir.resolve("src").resolve("main").resolve("java"));
            out.add(dir.resolve("src").resolve("test").resolve("java"));
            out.add(dir.resolve("src"));
            if (dir.equals(root)) break;
            dir = dir.getParent();
        }
        return new ArrayList<>(out);
    }

    static final class Resolver {
        private final Path root;
        private final List<Path> roots;

        Resolver(Path root, List<Path> roots) {
            this.root = root;
            this.roots = roots;
        }

        /** Package name of dir if it is inside the root (after symlinks) and not third-party. */
        private String internalPackage(Path dir) {
            Path real;
            try {
                real = dir.toRealPath();
            } catch (IOException e) {
                return null;
            }
            if (!real.startsWith(root)) return null;
            for (Path part : root.relativize(real)) {
                if (THIRD_PARTY.contains(part.toString())) return null;
            }
            return packageName(root, real);
        }

        /** Dotted type name, possibly nested: longest prefix first (module, then enclosing module). */
        String resolveType(List<String> segs) {
            for (int n = segs.size(); n >= 1; n--) {
                String rel = String.join("/", segs.subList(0, n)) + ".java";
                for (Path sr : roots) {
                    Path p = sr.resolve(rel);
                    if (!Files.isRegularFile(p)) continue;
                    try {
                        String pkg = internalPackage(p.toRealPath().getParent());
                        if (pkg != null) return pkg;
                    } catch (IOException ignored) {
                        // unreadable candidate: keep searching
                    }
                }
            }
            return null;
        }

        /** On-demand import: a directory that directly holds at least one .java file. */
        String resolvePackage(List<String> segs) {
            for (Path sr : roots) {
                Path d = sr.resolve(String.join("/", segs));
                if (!Files.isDirectory(d) || !hasJava(d)) continue;
                String pkg = internalPackage(d);
                if (pkg != null) return pkg;
            }
            return null;
        }

        private static boolean hasJava(Path dir) {
            try (Stream<Path> s = Files.list(dir)) {
                return s.anyMatch(p -> p.getFileName().toString().endsWith(".java") && !Files.isDirectory(p));
            } catch (IOException e) {
                return false;
            }
        }
    }

    // ----------------------------------------------------------------- JSON

    static String toJson(String pkg, Set<String> deps) {
        StringBuilder sb = new StringBuilder("{\"package\":");
        quote(sb, pkg);
        sb.append(",\"dependencies\":[");
        boolean first = true;
        for (String d : deps) {
            if (!first) sb.append(',');
            first = false;
            quote(sb, d);
        }
        return sb.append("]}\n").toString();
    }

    private static void quote(StringBuilder sb, String s) {
        sb.append('"');
        for (int i = 0; i < s.length(); i++) {
            char c = s.charAt(i);
            switch (c) {
                case '"' -> sb.append("\\\"");
                case '\\' -> sb.append("\\\\");
                case '\n' -> sb.append("\\n");
                case '\r' -> sb.append("\\r");
                case '\t' -> sb.append("\\t");
                default -> {
                    if (c < 0x20) sb.append(String.format("\\u%04x", (int) c));
                    else sb.append(c);
                }
            }
        }
        sb.append('"');
    }
}