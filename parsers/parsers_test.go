package parsers

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// NOTE: these tests set environment variables (t.Setenv), so none of them may
// use t.Parallel(). Run them with -race too: GetASTs is concurrent.
//
// Instead of depending on python3, Go or Java, the tests use the test binary
// itself as a fake parser. When it is started with fakeParserEnv set, TestMain
// behaves like a parser following the ArchLens spec:
//
//	<parser> [extra args] -r <rootDir> <file>   ->   JSON on stdout
//
// What it prints depends on the base name of <file>:
//
//	fail         stderr "boom: cannot parse", exit code 3
//	badjson      stdout that is not JSON
//	empty        no output at all
//	warn         warning on stderr, valid JSON on stdout, exit code 0
//	echo         package = its working directory, dependencies = its own arguments
//	nodeps       JSON without a "dependencies" key
//	extrafields  JSON with a field the spec does not define
//	anything else   package "pkg/<name without extension>", dependencies ["dep/a", "dep/b"]

const fakeParserEnv = "ARCHLENS_FAKE_PARSER"

func TestMain(m *testing.M) {
	if os.Getenv(fakeParserEnv) == "1" {
		runFakeParser(os.Args[1:])
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func runFakeParser(args []string) {
	file := ""
	if len(args) > 0 {
		file = args[len(args)-1]
	}
	base := filepath.Base(file)

	switch base {
	case "fail":
		fmt.Fprintln(os.Stderr, "boom: cannot parse")
		os.Exit(3)
	case "badjson":
		fmt.Print("this is not json")
	case "empty":
		// print nothing
	case "warn":
		fmt.Fprintln(os.Stderr, "warning: deprecated syntax")
		fmt.Print(`{"package":"pkg/warn","dependencies":[]}`)
	case "echo":
		cwd, _ := os.Getwd()
		out, _ := json.Marshal(ASTNode{Package: cwd, Dependencies: args})
		fmt.Print(string(out))
	case "nodeps":
		fmt.Print(`{"package":"pkg/nodeps"}`)
	case "extrafields":
		fmt.Print(`{"package":"pkg/extra","dependencies":["x"],"version":2}`)
	default:
		name := strings.TrimSuffix(base, filepath.Ext(base))
		out, _ := json.Marshal(ASTNode{Package: "pkg/" + name, Dependencies: []string{"dep/a", "dep/b"}})
		fmt.Print(string(out))
	}
}

// ---------- helpers ----------

// fakeParser returns a runCommand that launches the fake parser, optionally
// with extra arguments in front of the spec's own flags. The program path is
// always quoted, so paths containing spaces work too. extraArgs are inserted
// verbatim and may contain their own quoting.
func fakeParser(t *testing.T, extraArgs ...string) string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	t.Setenv(fakeParserEnv, "1")
	return strings.Join(append([]string{`"` + exe + `"`}, extraArgs...), " ")
}

// byFile indexes results by file name and fails on duplicates.
func byFile(t *testing.T, results []ASTResult) map[string]ASTResult {
	t.Helper()
	m := make(map[string]ASTResult, len(results))
	for _, r := range results {
		if _, dup := m[r.File]; dup {
			t.Errorf("file %q returned more than once", r.File)
		}
		m[r.File] = r
	}
	return m
}

func resolve(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", p, err)
	}
	return r
}

// ---------- GetAST ----------

func Test_GetAST_ParsesParserOutput(t *testing.T) {
	got, err := GetAST("pkg/a.py", t.TempDir(), fakeParser(t))
	if err != nil {
		t.Fatalf("GetAST returned error: %v", err)
	}
	if got.Package != "pkg/a" {
		t.Errorf("Package = %q, want %q", got.Package, "pkg/a")
	}
	if want := []string{"dep/a", "dep/b"}; !slices.Equal(got.Dependencies, want) {
		t.Errorf("Dependencies = %v, want %v", got.Dependencies, want)
	}
}

func Test_GetAST_PassesArgumentsInSpecOrder(t *testing.T) {
	root := t.TempDir()

	got, err := GetAST("echo", root, fakeParser(t, "--flag", "value"))
	if err != nil {
		t.Fatalf("GetAST returned error: %v", err)
	}

	// The command's own arguments come first, then -r <rootDir> <file>
	want := []string{"--flag", "value", "-r", root, "echo"}
	if !slices.Equal(got.Dependencies, want) {
		t.Errorf("parser received args %v, want %v", got.Dependencies, want)
	}
}

func Test_GetAST_RunsInRootDir(t *testing.T) {
	root := t.TempDir()

	got, err := GetAST("echo", root, fakeParser(t))
	if err != nil {
		t.Fatalf("GetAST returned error: %v", err)
	}
	if resolve(t, got.Package) != resolve(t, root) {
		t.Errorf("parser ran in %q, want %q", got.Package, root)
	}
}

func Test_GetAST_QuotedArgumentsStayOneArgument(t *testing.T) {
	root := t.TempDir()

	got, err := GetAST("echo", root, fakeParser(t, `--name="two words"`, `'single quoted'`))
	if err != nil {
		t.Fatalf("GetAST returned error: %v", err)
	}

	want := []string{"--name=two words", "single quoted", "-r", root, "echo"}
	if !slices.Equal(got.Dependencies, want) {
		t.Errorf("parser received args %q, want %q", got.Dependencies, want)
	}
}

func Test_GetAST_ProgramPathWithSpaces(t *testing.T) {
	t.Setenv(fakeParserEnv, "1")

	// Copy the test binary into a directory whose name contains a space
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	data, err := os.ReadFile(exe)
	if err != nil {
		t.Fatalf("read test binary: %v", err)
	}
	dir := filepath.Join(t.TempDir(), "my tools")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	program := filepath.Join(dir, filepath.Base(exe)) // keeps .exe on Windows
	if err := os.WriteFile(program, data, 0o755); err != nil {
		t.Fatalf("copy test binary: %v", err)
	}

	for name, runCommand := range map[string]string{
		"double quotes": `"` + program + `"`,
		"single quotes": `'` + program + `'`,
	} {
		t.Run(name, func(t *testing.T) {
			got, err := GetAST("a.py", t.TempDir(), runCommand)
			if err != nil {
				t.Fatalf("GetAST returned error: %v", err)
			}
			if got.Package != "pkg/a" {
				t.Errorf("Package = %q, want %q", got.Package, "pkg/a")
			}
		})
	}
}

func Test_GetAST_UnterminatedQuote(t *testing.T) {
	got, err := GetAST("a.py", t.TempDir(), `python3 "my dir/arch.py`)
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "unterminated") {
		t.Errorf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("got %+v, want nil", got)
	}
}

func Test_GetAST_EmptyRunCommand(t *testing.T) {
	for _, rc := range []string{"", "   ", "\t\n"} {
		t.Run(fmt.Sprintf("%q", rc), func(t *testing.T) {
			got, err := GetAST("a.py", t.TempDir(), rc)
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), "runCommand is empty") {
				t.Errorf("unexpected error: %v", err)
			}
			if got != nil {
				t.Errorf("got %+v, want nil", got)
			}
		})
	}
}

func Test_GetAST_CommandFailureIncludesStderr(t *testing.T) {
	got, err := GetAST("fail", t.TempDir(), fakeParser(t))
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{"command failed", "boom: cannot parse"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should contain %q", err, want)
		}
	}
	if got != nil {
		t.Errorf("got %+v, want nil", got)
	}
}

func Test_GetAST_InvalidOutput(t *testing.T) {
	for _, file := range []string{"badjson", "empty"} {
		t.Run(file, func(t *testing.T) {
			got, err := GetAST(file, t.TempDir(), fakeParser(t))
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), "bad json") {
				t.Errorf("unexpected error: %v", err)
			}
			var syntaxErr *json.SyntaxError
			if !errors.As(err, &syntaxErr) {
				t.Errorf("error should wrap a *json.SyntaxError, got: %v", err)
			}
			if got != nil {
				t.Errorf("got %+v, want nil", got)
			}
		})
	}
}

func Test_GetAST_IgnoresStderrOnSuccess(t *testing.T) {
	got, err := GetAST("warn", t.TempDir(), fakeParser(t))
	if err != nil {
		t.Fatalf("warnings on stderr must not fail the parse, got: %v", err)
	}
	if got.Package != "pkg/warn" {
		t.Errorf("Package = %q, want %q", got.Package, "pkg/warn")
	}
}

func Test_GetAST_ToleratesMissingAndUnknownFields(t *testing.T) {
	t.Run("missing dependencies", func(t *testing.T) {
		got, err := GetAST("nodeps", t.TempDir(), fakeParser(t))
		if err != nil {
			t.Fatalf("GetAST returned error: %v", err)
		}
		if got.Package != "pkg/nodeps" || len(got.Dependencies) != 0 {
			t.Errorf("got %+v, want package pkg/nodeps and no dependencies", got)
		}
	})

	t.Run("unknown fields are ignored", func(t *testing.T) {
		got, err := GetAST("extrafields", t.TempDir(), fakeParser(t))
		if err != nil {
			t.Fatalf("GetAST returned error: %v", err)
		}
		if got.Package != "pkg/extra" || !slices.Equal(got.Dependencies, []string{"x"}) {
			t.Errorf("got %+v", got)
		}
	})
}

func Test_GetAST_ProgramNotFound(t *testing.T) {
	got, err := GetAST("a.py", t.TempDir(), "archlens-no-such-binary --flag")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !errors.Is(err, exec.ErrNotFound) {
		t.Errorf("error should wrap exec.ErrNotFound, got: %v", err)
	}
	if got != nil {
		t.Errorf("got %+v, want nil", got)
	}
}

func Test_GetAST_MissingRootDir(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")

	got, err := GetAST("a.py", missing, fakeParser(t))
	if err == nil {
		t.Fatal("expected an error when the root directory does not exist")
	}
	if got != nil {
		t.Errorf("got %+v, want nil", got)
	}
}

// ---------- GetASTs ----------

func Test_GetASTs_ParsesEveryFileOnce(t *testing.T) {
	// More files than workers, so every worker handles several jobs
	n := max(30, 3*runtime.NumCPU()+1)
	files := make([]string, n)
	for i := range files {
		files[i] = fmt.Sprintf("src/f%03d.py", i)
	}

	results := GetASTs(files, t.TempDir(), fakeParser(t))

	if len(results) != n {
		t.Fatalf("got %d results, want %d", len(results), n)
	}
	got := byFile(t, results)
	for i, f := range files {
		r, ok := got[f]
		if !ok {
			t.Errorf("no result for %q", f)
			continue
		}
		if r.Err != nil {
			t.Errorf("%s: unexpected error: %v", f, r.Err)
			continue
		}
		if want := fmt.Sprintf("pkg/f%03d", i); r.AST == nil || r.AST.Package != want {
			t.Errorf("%s: AST = %+v, want package %q", f, r.AST, want)
		}
	}
}

func Test_GetASTs_KeepsFileNamesAsGiven(t *testing.T) {
	files := []string{"./a.py", "dir/b.py", "dir/sub/c.py"}

	got := byFile(t, GetASTs(files, t.TempDir(), fakeParser(t)))

	for _, f := range files {
		if r, ok := got[f]; !ok || r.Err != nil {
			t.Errorf("missing or failed result for %q: %+v", f, r)
		}
	}
}

func Test_GetASTs_SingleFile(t *testing.T) {
	results := GetASTs([]string{"only.py"}, t.TempDir(), fakeParser(t))

	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Err != nil || results[0].AST == nil || results[0].AST.Package != "pkg/only" {
		t.Errorf("unexpected result: %+v", results[0])
	}
}

func Test_GetASTs_NoFiles(t *testing.T) {
	for name, files := range map[string][]string{"nil": nil, "empty": {}} {
		t.Run(name, func(t *testing.T) {
			if results := GetASTs(files, t.TempDir(), fakeParser(t)); len(results) != 0 {
				t.Errorf("got %d results, want 0", len(results))
			}
		})
	}
}

func Test_GetASTs_CapturesErrorsPerFile(t *testing.T) {
	files := []string{"ok1.py", "fail", "ok2.py", "badjson", "empty", "ok3.py"}
	wantOK := map[string]bool{"ok1.py": true, "ok2.py": true, "ok3.py": true}

	results := GetASTs(files, t.TempDir(), fakeParser(t))

	// A failing file must not abort the batch
	if len(results) != len(files) {
		t.Fatalf("got %d results, want %d", len(results), len(files))
	}
	got := byFile(t, results)
	for _, f := range files {
		r, ok := got[f]
		if !ok {
			t.Errorf("no result for %q", f)
			continue
		}
		if wantOK[f] {
			if r.Err != nil || r.AST == nil {
				t.Errorf("%s: want success, got AST=%+v err=%v", f, r.AST, r.Err)
			}
		} else if r.Err == nil || r.AST != nil {
			t.Errorf("%s: want error and nil AST, got AST=%+v err=%v", f, r.AST, r.Err)
		}
	}
}

func Test_GetASTs_EmptyRunCommandFailsEveryFile(t *testing.T) {
	files := []string{"a.py", "b.py", "c.py"}

	results := GetASTs(files, t.TempDir(), "")

	if len(results) != len(files) {
		t.Fatalf("got %d results, want %d", len(results), len(files))
	}
	for _, r := range results {
		if r.AST != nil {
			t.Errorf("%s: AST should be nil, got %+v", r.File, r.AST)
		}
		if r.Err == nil || !strings.Contains(r.Err.Error(), "runCommand is empty") {
			t.Errorf("%s: unexpected error: %v", r.File, r.Err)
		}
	}
}

// ---------- splitCommand ----------

func Test_splitCommand(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"single word", "python3", []string{"python3"}},
		{"two words", "python3 arch.py", []string{"python3", "arch.py"}},
		{"extra whitespace", "  python3   arch.py  ", []string{"python3", "arch.py"}},
		{"tabs and newlines", "python3\tarch.py\n--fast", []string{"python3", "arch.py", "--fast"}},
		{"empty string", "", nil},
		{"only whitespace", " \t\n ", nil},

		{"double quotes group whitespace", `python3 "my dir/arch.py"`, []string{"python3", "my dir/arch.py"}},
		{"single quotes group whitespace", `python3 'my dir/arch.py'`, []string{"python3", "my dir/arch.py"}},
		{"quoted program", `"/opt/my tools/parser" -x`, []string{"/opt/my tools/parser", "-x"}},
		{"quote starting mid-word", `--name="two words"`, []string{"--name=two words"}},
		{"quotes in the middle of a word", `a"b c"d`, []string{"ab cd"}},
		{"other quote kind is literal inside quotes", `"it's" fine`, []string{"it's", "fine"}},
		{"double quotes inside single quotes", `'say "hi"'`, []string{`say "hi"`}},
		{"empty quotes make an empty argument", `a "" b`, []string{"a", "", "b"}},
		{"only empty quotes", `""`, []string{""}},
		{"whitespace inside quotes is preserved", `"  a  "`, []string{"  a  "}},

		{"backslashes are not escapes", `C:\tools\parser.exe -x`, []string{`C:\tools\parser.exe`, "-x"}},
		{
			"quoted Windows path with spaces",
			`"C:\Program Files\Python\python.exe" arch.py`,
			[]string{`C:\Program Files\Python\python.exe`, "arch.py"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := splitCommand(tt.input)
			if err != nil {
				t.Fatalf("splitCommand(%q) returned error: %v", tt.input, err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("splitCommand(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func Test_splitCommand_UnterminatedQuote(t *testing.T) {
	for _, input := range []string{`python3 "arch.py`, `python3 'arch.py`, `"`, `a "b 'c'`} {
		t.Run(input, func(t *testing.T) {
			got, err := splitCommand(input)
			if err == nil {
				t.Fatalf("expected an error, got %q", got)
			}
			if !strings.Contains(err.Error(), "unterminated") {
				t.Errorf("unexpected error: %v", err)
			}
			if got != nil {
				t.Errorf("got %q, want nil on error", got)
			}
		})
	}
}

// ---------- workerCount ----------

func Test_workerCount(t *testing.T) {
	cpus := runtime.NumCPU()

	tests := []struct {
		name string
		jobs int
		want int
	}{
		{"no jobs need no workers", 0, 0},
		{"one job needs one worker", 1, 1},
		{"fewer jobs than CPUs", cpus - 1, cpus - 1},
		{"as many jobs as CPUs", cpus, cpus},
		{"one more job than CPUs is capped", cpus + 1, cpus},
		{"many jobs are capped at the CPU count", cpus * 10, cpus},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := workerCount(tt.jobs); got != tt.want {
				t.Errorf("workerCount(%d) = %d, want %d", tt.jobs, got, tt.want)
			}
		})
	}
}