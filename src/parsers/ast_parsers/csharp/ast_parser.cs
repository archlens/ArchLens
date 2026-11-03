using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Text;
using Microsoft.CodeAnalysis;
using Microsoft.CodeAnalysis.CSharp;
using Microsoft.CodeAnalysis.CSharp.Syntax;

namespace ArchLens.Parser
{
    public sealed class ParserException : Exception
    {
        public ParserException(string message) : base(message) { }
    }

    public record PathInfo(string FullPath);

    public static class AstParser
    {
        private static readonly HashSet<string> ThirdParty = new(StringComparer.OrdinalIgnoreCase)
        {
            "bin", "obj", "packages", "node_modules", "vendor", ".git", ".vs"
        };

        public static void Main(string[] args)
        {
            try
            {
                string output = Run(args);
                using var stdout = new StreamWriter(Console.OpenStandardOutput(), new UTF8Encoding(false));
                stdout.Write(output);
            }
            catch (ParserException ex)
            {
                Console.Error.WriteLine($"csharpparser: {ex.Message}");
                Environment.Exit(1);
            }
            catch (Exception ex)
            {
                Console.Error.WriteLine($"csharpparser: {ex.Message}");
                Environment.Exit(1);
            }
        }

        public static string Run(string[] args)
        {
            string? fileArg = null;
            string rootArg = ".";

            for (int i = 0; i < args.Length; i++)
            {
                string a = args[i];
                if (a == "-r" || a == "--root")
                {
                    if (i + 1 >= args.Length) throw new ParserException($"{a} requires a directory");
                    rootArg = args[++i];
                }
                else if (a.StartsWith("--root="))
                {
                    rootArg = a.Substring("--root=".Length);
                }
                else if (a.StartsWith("-r="))
                {
                    rootArg = a.Substring("-r=".Length);
                }
                else if (a.StartsWith("-") && a.Length > 1)
                {
                    throw new ParserException($"unknown option '{a}'");
                }
                else if (fileArg != null)
                {
                    throw new ParserException("expected exactly one file");
                }
                else
                {
                    fileArg = a;
                }
            }

            if (fileArg == null) throw new ParserException("usage: csharpparser <file> [-r DIR | --root DIR]");
            if (string.IsNullOrEmpty(rootArg)) throw new ParserException("root must not be empty");

            PathInfo rootInfo;
            try
            {
                var fullRoot = Path.GetFullPath(rootArg);
                var realRoot = new DirectoryInfo(fullRoot).ResolveLinkTarget(true)?.FullName ?? fullRoot;
                rootInfo = new PathInfo(realRoot);
            }
            catch (Exception e)
            {
                throw new ParserException($"root '{rootArg}' is not usable: {e.Message}");
            }

            if (!Directory.Exists(rootInfo.FullPath))
            {
                throw new ParserException($"root '{rootArg}' is not a directory");
            }

            PathInfo fileInfo;
            try
            {
                var fullFile = Path.GetFullPath(fileArg);
                var realFile = new FileInfo(fullFile).ResolveLinkTarget(true)?.FullName ?? fullFile;
                fileInfo = new PathInfo(realFile);
            }
            catch (Exception e)
            {
                throw new ParserException($"cannot read '{fileArg}': {e.Message}");
            }

            if (!File.Exists(fileInfo.FullPath))
            {
                throw new ParserException($"'{fileArg}' is not a readable file");
            }

            var fileDir = Path.GetDirectoryName(fileInfo.FullPath)!;
            if (!fileDir.StartsWith(rootInfo.FullPath, StringComparison.OrdinalIgnoreCase))
            {
                throw new ParserException($"file '{fileArg}' is outside the root '{rootArg}'");
            }

            if (!fileInfo.FullPath.EndsWith(".cs", StringComparison.OrdinalIgnoreCase))
            {
                throw new ParserException($"'{fileArg}' is not a .cs file");
            }

            var parsed = ParseFile(fileInfo.FullPath);

            var searchRoots = GetSearchRoots(rootInfo.FullPath, fileDir);
            var resolver = new Resolver(rootInfo.FullPath, searchRoots);
            string ownPackage = PackageName(rootInfo.FullPath, fileDir);

            var deps = new List<string>();
            foreach (var imp in parsed.Imports)
            {
                List<string> segs = imp.Segments;
                if (imp.IsStatic)
                {
                    if (!imp.IsStar && segs.Count >= 2)
                    {
                        segs = segs.Take(segs.Count - 1).ToList(); // drop member name
                    }
                }

                string? pkg = resolver.Resolve(segs);
                if (pkg != null && pkg != ownPackage && !deps.Contains(pkg))
                {
                    deps.Add(pkg);
                }
            }

            return ToJson(ownPackage, deps);
        }

        // ------------------------------------------------------------- Parsing

        public record ImportInfo(bool IsStatic, bool IsStar, List<String> Segments);
        public record ParsedFile(List<ImportInfo> Imports);

private static ParsedFile ParseFile(string filePath)
        {
            string code;
            try
            {
                code = File.ReadAllText(filePath);
            }
            catch (IOException e)
            {
                throw new ParserException($"cannot read '{filePath}': {e.Message}");
            }

            SyntaxTree tree = CSharpSyntaxTree.ParseText(code, path: filePath);
            var diagnostics = tree.GetDiagnostics().Where(d => d.Severity == DiagnosticSeverity.Error).ToList();
            if (diagnostics.Count > 0)
            {
                var d = diagnostics[0];
                var lineSpan = d.Location.GetLineSpan();
                throw new ParserException($"{filePath}:{lineSpan.StartLinePosition.Line + 1}: syntax error: {d.GetMessage()}");
            }

            var rootNode = tree.GetRoot();
            var imports = new List<ImportInfo>();

            // All using directives (including global usings, which have the GlobalKeyword set)
            foreach (var usingDirective in rootNode.DescendantNodes().OfType<UsingDirectiveSyntax>())
            {
                ProcessUsing(usingDirective.Name, usingDirective.StaticKeyword, imports);
            }

            return new ParsedFile(imports);
        }

        private static void ProcessUsing(NameSyntax? nameSyntax, SyntaxToken staticKeyword, List<ImportInfo> imports)
        {
            if (nameSyntax == null) return;

            var segs = GetSegments(nameSyntax);
            bool isStatic = staticKeyword.IsKind(SyntaxKind.StaticKeyword);
            bool isStar = false; 

            if (segs.Count > 0 && segs[^1] == "*")
            {
                isStar = true;
                segs = segs.Take(segs.Count - 1).ToList();
            }

            if (segs.Count > 0)
            {
                imports.Add(new ImportInfo(isStatic, isStar, segs));
            }
        }

        private static List<string> GetSegments(NameSyntax nameSyntax)
        {
            var segments = new List<string>();
            FlattenName(nameSyntax, segments);
            return segments;
        }

        private static void FlattenName(SyntaxNode syntax, List<string> segments)
        {
            if (syntax is IdentifierNameSyntax id)
            {
                segments.Add(id.Identifier.Text);
            }
            else if (syntax is QualifiedNameSyntax qual)
            {
                FlattenName(qual.Left, segments);
                FlattenName(qual.Right, segments);
            }
            else if (syntax is AliasQualifiedNameSyntax alias)
            {
                segments.Add(alias.Alias.Identifier.Text);
                FlattenName(alias.Name, segments);
            }
        }

        // ------------------------------------------------------------ Resolving

        private static string PackageName(string root, string dir)
        {
            var rel = Path.GetRelativePath(root, dir);
            if (rel == "." || string.IsNullOrEmpty(rel)) return ".";
            return rel.Replace('\\', '/');
        }

        private static List<string> GetSearchRoots(string root, string fileDir)
        {
            var outRoots = new LinkedHashSet<string>();
            var dir = fileDir;
            while (dir != null)
            {
                outRoots.Add(dir);
                if (string.Equals(dir, root, StringComparison.OrdinalIgnoreCase)) break;
                dir = Path.GetDirectoryName(dir);
            }
            return outRoots.ToList();
        }

        private sealed class Resolver
        {
            private readonly string root;
            private readonly List<string> searchRoots;

            public Resolver(string root, List<string> searchRoots)
            {
                this.root = root;
                this.searchRoots = searchRoots;
            }

            private string? InternalPackage(string dir)
            {
                try
                {
                    var info = new DirectoryInfo(dir);
                    if (!info.Exists) return null;
                    var real = info.ResolveLinkTarget(true)?.FullName ?? info.FullName;

                    if (!real.StartsWith(root, StringComparison.OrdinalIgnoreCase)) return null;

                    var rel = Path.GetRelativePath(root, real);
                    if (rel != "." && !string.IsNullOrEmpty(rel))
                    {
                        var parts = rel.Split(Path.DirectorySeparatorChar, Path.AltDirectorySeparatorChar);
                        foreach (var part in parts)
                        {
                            if (ThirdParty.Contains(part)) return null;
                        }
                    }

                    return PackageName(root, real);
                }
                catch
                {
                    return null;
                }
            }

            public string? Resolve(List<string> segs)
            {
                // Support both direct matching and stripping root project namespace prefix (e.g. Archlens.Domain -> Domain)
                var candidateSegmentLists = new List<List<string>> { segs };
                if (segs.Count > 1)
                {
                    candidateSegmentLists.Add(segs.Skip(1).ToList());
                }

                foreach (var currentSegs in candidateSegmentLists)
                {
                    for (int n = currentSegs.Count; n >= 1; n--)
                    {
                        var relParts = currentSegs.Take(n).ToArray();
                        var relDir = Path.Combine(relParts);

                        foreach (var sr in searchRoots)
                        {
                            var candidateDir = Path.Combine(sr, relDir);
                            if (Directory.Exists(candidateDir))
                            {
                                var pkg = InternalPackage(candidateDir);
                                if (pkg != null) return pkg;
                            }
                        }
                    }
                }
                return null;
            }
        }

        // ----------------------------------------------------------------- JSON

        private static string ToJson(string pkg, List<string> deps)
        {
            var sb = new StringBuilder("{\"package\":");
            Quote(sb, pkg);
            sb.Append(",\"dependencies\":[");
            bool first = true;
            foreach (var d in deps)
            {
                if (!first) sb.Append(',');
                first = false;
                Quote(sb, d);
            }
            return sb.Append("]}\n").ToString();
        }

        private static void Quote(StringBuilder sb, string s)
        {
            sb.Append('"');
            foreach (char c in s)
            {
                switch (c)
                {
                    case '"': sb.Append("\\\""); break;
                    case '\\': sb.Append("\\\\"); break;
                    case '\n': sb.Append("\\n"); break;
                    case '\r': sb.Append("\\r"); break;
                    case '\t': sb.Append("\\t"); break;
                    default:
                        if (c < 0x20) sb.Append($"\\u{(int)c:x4}");
                        else sb.Append(c);
                        break;
                }
            }
            sb.Append('"');
        }
    }

    internal class LinkedHashSet<T> where T : notnull
    {
        private readonly HashSet<T> _set = new();
        private readonly List<T> _list = new();

        public bool Add(T item)
        {
            if (_set.Add(item))
            {
                _list.Add(item);
                return true;
            }
            return false;
        }

        public List<T> ToList() => _list;
    }
}