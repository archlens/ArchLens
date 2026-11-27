package input

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/archlens/archlens/utils"
	"go.uber.org/zap"
)

// NOTE: these tests change the working directory (GetFiles does it itself and
// matchFiles needs it), so none of them may use t.Parallel().

func TestMain(m *testing.M) {
	// logger is captured from utils.Sugar at package init; make sure it is usable
	utils.Sugar = zap.NewNop().Sugar()
	os.Exit(m.Run())
}

// ---------- helpers ----------

// writeFiles creates the given slash-separated relative paths under root.
func writeFiles(t *testing.T, root string, paths ...string) {
	t.Helper()
	for _, p := range paths {
		full := filepath.Join(root, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", p, err)
		}
		if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
	}
}

// writeConfig writes content to a fresh archlens.json and returns its path.
func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "archlens.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

// slashSorted normalizes paths to forward slashes and sorts them, so results
// can be compared regardless of OS and map iteration order.
func slashSorted(paths []string) []string {
	out := make([]string, len(paths))
	for i, p := range paths {
		out[i] = filepath.ToSlash(p)
	}
	slices.Sort(out)
	return out
}

func keys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return slashSorted(out)
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	restore := utils.WithWorkingDirectory(dir)
	t.Cleanup(restore)
}

// checkCacheExclude verifies that Load kept the configured excludes, in order,
// and added exactly one entry that excludes the cache directory (without
// depending on the exact spelling of that entry) but nothing else.
func checkCacheExclude(t *testing.T, name string, got View, configured []string) {
	t.Helper()
	if len(got.Exclude) != len(configured)+1 {
		t.Errorf("view %q Exclude = %v, want %v plus one cache entry", name, got.Exclude, configured)
		return
	}
	if !slices.Equal(got.Exclude[:len(configured)], configured) {
		t.Errorf("view %q Exclude = %v, want it to start with %v", name, got.Exclude, configured)
	}
	cacheOnly := View{Include: []string{"."}, Exclude: got.Exclude[len(configured):]}
	if cacheOnly.Match(".archlens/sub") {
		t.Errorf("view %q: added entry %q does not exclude the cache directory", name, got.Exclude[len(configured):])
	}
	if !cacheOnly.Match("src") {
		t.Errorf("view %q: added entry %q excludes more than the cache directory", name, got.Exclude[len(configured):])
	}
}

// ---------- Load ----------

func TestLoad_ParsesAllFields(t *testing.T) {
	path := writeConfig(t, `{
		"name": "demo",
		"github": {"url": "https://github.com/archlens/demo", "branch": "main"},
		"rootFolder": "src",
		"files": ["*.py", "*.pyi"],
		"saveLocation": "docs/arch",
		"runCommand": "python3 ast_parser.py",
		"views": {
			"complete": {"include": ["pkg", "app/**"], "exclude": ["venv"]}
		}
	}`)

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if got.Name != "demo" {
		t.Errorf("Name = %q, want %q", got.Name, "demo")
	}
	if got.Github.Url != "https://github.com/archlens/demo" || got.Github.Branch != "main" {
		t.Errorf("Github = %+v", got.Github)
	}
	if got.RootFolder != "src" {
		t.Errorf("RootFolder = %q, want %q", got.RootFolder, "src")
	}
	if !slices.Equal(got.Files, []string{"*.py", "*.pyi"}) {
		t.Errorf("Files = %v, want [*.py *.pyi]", got.Files)
	}
	if got.SaveLocation != "docs/arch" {
		t.Errorf("SaveLocation = %q, want %q", got.SaveLocation, "docs/arch")
	}
	if got.RunCommand != "python3 ast_parser.py" {
		t.Errorf("RunCommand = %q", got.RunCommand)
	}

	view, ok := got.Views["complete"]
	if !ok {
		t.Fatal("view \"complete\" missing")
	}
	if !slices.Equal(view.Include, []string{"pkg", "app/**"}) {
		t.Errorf("Include = %v", view.Include)
	}
	checkCacheExclude(t, "complete", view, []string{"venv"})
}

func TestLoad_AppendsCacheExclusionToEveryView(t *testing.T) {
	path := writeConfig(t, `{"views": {
		"noExclude":    {"include": ["cmd"]},
		"emptyExclude": {"include": ["cmd"], "exclude": []},
		"withExclude":  {"include": ["cmd"], "exclude": ["vendor", "**/testdata"]}
	}}`)

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	want := map[string][]string{
		"noExclude":    nil,
		"emptyExclude": {},
		"withExclude":  {"vendor", "**/testdata"},
	}
	if len(got.Views) != len(want) {
		t.Fatalf("got %d views, want %d", len(got.Views), len(want))
	}
	for name, configured := range want {
		view, ok := got.Views[name]
		if !ok {
			t.Errorf("view %q missing", name)
			continue
		}
		checkCacheExclude(t, name, view, configured)
		if !slices.Equal(view.Include, []string{"cmd"}) {
			t.Errorf("view %q Include = %v, want [cmd]", name, view.Include)
		}
	}
}

func TestLoad_NoViews(t *testing.T) {
	path := writeConfig(t, `{"name": "bare"}`)

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got.Name != "bare" {
		t.Errorf("Name = %q, want %q", got.Name, "bare")
	}
	if len(got.Views) != 0 {
		t.Errorf("Views = %v, want empty", got.Views)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	got, err := Load(filepath.Join(t.TempDir(), "nope.json"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("error = %v, want one wrapping os.ErrNotExist", err)
	}
	if got != nil {
		t.Errorf("got %+v, want nil", got)
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	path := writeConfig(t, `{not json`)

	got, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if got != nil {
		t.Errorf("got %+v, want nil", got)
	}
}

// ---------- View.Match ----------

func TestView_Match(t *testing.T) {
	tests := []struct {
		name string
		view View
		dir  string
		want bool
	}{
		// include
		{"package matches itself", View{Include: []string{"cmd"}}, "cmd", true},
		{"package matches subpackages", View{Include: []string{"cmd"}}, "cmd/x/y", true},
		{"package is not a prefix match", View{Include: []string{"cmd"}}, "cmdx", false},
		{"package does not match its parent", View{Include: []string{"cmd/sub"}}, "cmd", false},
		{"package does not match siblings", View{Include: []string{"cmd"}}, "export", false},
		{"package does not match the root", View{Include: []string{"cmd"}}, ".", false},
		{"dot matches the root", View{Include: []string{"."}}, ".", true},
		{"dot matches everything", View{Include: []string{"."}}, "a/b/c", true},
		{"trailing slash is ignored", View{Include: []string{"cmd/"}}, "cmd/x", true},
		{"leading ./ is ignored", View{Include: []string{"./cmd"}}, "cmd", true},
		{"dir is cleaned", View{Include: []string{"cmd"}}, "./cmd/", true},
		{"glob include matches subpackages", View{Include: []string{"internal/*"}}, "internal/a", true},
		{"glob include matches below them", View{Include: []string{"internal/*"}}, "internal/a/b", true},
		{"glob include does not match its base", View{Include: []string{"internal/*"}}, "internal", false},
		{"any include is enough", View{Include: []string{"cmd", "export"}}, "export/html", true},
		{"no include matches nothing", View{}, "cmd", false},
		{"exclude without include matches nothing", View{Exclude: []string{"vendor"}}, "cmd", false},

		// exclude
		{"exclude removes the package", View{Include: []string{"."}, Exclude: []string{"export/html"}}, "export/html", false},
		{"exclude removes subpackages", View{Include: []string{"."}, Exclude: []string{"export/html"}}, "export/html/deep", false},
		{"exclude keeps the parent", View{Include: []string{"."}, Exclude: []string{"export/html"}}, "export", true},
		{"exclude is not a prefix match", View{Include: []string{"."}, Exclude: []string{"export/html"}}, "export/htmlx", true},
		{"exclude wins over include", View{Include: []string{"cmd"}, Exclude: []string{"cmd"}}, "cmd", false},
		{"exclude can carve out a subpackage", View{Include: []string{"cmd"}, Exclude: []string{"cmd/sub"}}, "cmd/sub/x", false},
		{"glob exclude matches at any depth", View{Include: []string{"."}, Exclude: []string{"**/testdata"}}, "a/b/testdata", false},
		{"glob exclude matches at the top", View{Include: []string{"."}, Exclude: []string{"**/testdata"}}, "testdata", false},
		{"glob exclude below a match", View{Include: []string{"."}, Exclude: []string{"**/testdata"}}, "a/testdata/x", false},
		{"glob exclude keeps other packages", View{Include: []string{"."}, Exclude: []string{"**/testdata"}}, "a/b", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.view.Match(tt.dir); got != tt.want {
				t.Errorf("%+v.Match(%q) = %v, want %v", tt.view, tt.dir, got, tt.want)
			}
		})
	}
}

// ---------- matchFiles ----------

func TestMatchFiles(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root,
		"main.py", "utils.py", "README.md",
		"pkg/models.py", "pkg/__init__.py", "pkg/sub/deep.py",
		"venv/lib/site.py",
	)
	chdir(t, root)

	tests := []struct {
		name     string
		patterns []string
		want     []string
	}{
		{
			name:     "top-level glob",
			patterns: []string{"*.py"},
			want:     []string{"main.py", "utils.py"},
		},
		{
			name:     "recursive glob",
			patterns: []string{"**/*.py"},
			want: []string{
				"main.py", "utils.py",
				"pkg/__init__.py", "pkg/models.py", "pkg/sub/deep.py",
				"venv/lib/site.py",
			},
		},
		{
			name:     "recursive glob scoped to a directory",
			patterns: []string{"pkg/**/*.py"},
			want:     []string{"pkg/__init__.py", "pkg/models.py", "pkg/sub/deep.py"},
		},
		{
			name:     "multiple patterns are unioned",
			patterns: []string{"*.py", "*.md"},
			want:     []string{"README.md", "main.py", "utils.py"},
		},
		{
			name:     "overlapping patterns are deduplicated",
			patterns: []string{"*.py", "main.*"},
			want:     []string{"main.py", "utils.py"},
		},
		{
			name:     "directories are not returned",
			patterns: []string{"pkg"},
			want:     nil,
		},
		{
			name:     "no matches",
			patterns: []string{"*.rs"},
			want:     nil,
		},
		{
			name:     "no patterns",
			patterns: nil,
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := matchFiles(tt.patterns)
			if err != nil {
				t.Fatalf("matchFiles returned error: %v", err)
			}
			if !slices.Equal(keys(got), slashSorted(tt.want)) {
				t.Errorf("matchFiles(%v) = %v, want %v", tt.patterns, keys(got), slashSorted(tt.want))
			}
		})
	}
}

func TestMatchFiles_MalformedPattern(t *testing.T) {
	chdir(t, t.TempDir())

	if _, err := matchFiles([]string{"["}); err == nil {
		t.Fatal("expected error for malformed pattern")
	}
}

// ---------- GetFiles ----------

func TestGetFiles(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root,
		"main.go", "main_test.go",
		"internal/app.go", "internal/app_test.go",
		"vendor/lib/lib.go",
		"docs/notes.md",
		"cmd/run.go", "cmd/sub/deep.go",
		"export/export.go", "export/html/page.go",
	)

	goExts := []string{"*.go"}
	allGo := []string{
		"main.go", "main_test.go",
		"internal/app.go", "internal/app_test.go",
		"vendor/lib/lib.go",
		"cmd/run.go", "cmd/sub/deep.go",
		"export/export.go", "export/html/page.go",
	}

	tests := []struct {
		name string
		view View
		exts []string // goExts when nil
		want []string
	}{
		{
			name: "dot selects the whole tree",
			view: View{Include: []string{"."}},
			want: allGo,
		},
		{
			name: "package is searched recursively",
			view: View{Include: []string{"cmd"}},
			want: []string{"cmd/run.go", "cmd/sub/deep.go"},
		},
		{
			name: "package does not include the root package",
			view: View{Include: []string{"internal"}},
			want: []string{"internal/app.go", "internal/app_test.go"},
		},
		{
			name: "several packages are unioned",
			view: View{Include: []string{"cmd", "export"}},
			want: []string{"cmd/run.go", "cmd/sub/deep.go", "export/export.go", "export/html/page.go"},
		},
		{
			name: "overlapping packages give each file once",
			view: View{Include: []string{"cmd", "cmd/sub"}},
			want: []string{"cmd/run.go", "cmd/sub/deep.go"},
		},
		{
			name: "glob include",
			view: View{Include: []string{"export/*"}},
			want: []string{"export/html/page.go"},
		},
		{
			name: "only files with a configured extension",
			view: View{Include: []string{"."}},
			want: allGo, // docs/notes.md is not here
		},
		{
			name: "several extensions",
			view: View{Include: []string{"docs", "cmd"}},
			exts: []string{"*.go", "*.md"},
			want: []string{"docs/notes.md", "cmd/run.go", "cmd/sub/deep.go"},
		},
		{
			name: "other extension",
			view: View{Include: []string{"."}},
			exts: []string{"*.md"},
			want: []string{"docs/notes.md"},
		},
		{
			name: "exclude removes a package and its subpackages",
			view: View{Include: []string{"."}, Exclude: []string{"cmd"}},
			want: []string{
				"main.go", "main_test.go",
				"internal/app.go", "internal/app_test.go",
				"vendor/lib/lib.go",
				"export/export.go", "export/html/page.go",
			},
		},
		{
			name: "exclude a nested package keeps its parent",
			view: View{Include: []string{"."}, Exclude: []string{"export/html", "vendor"}},
			want: []string{
				"main.go", "main_test.go",
				"internal/app.go", "internal/app_test.go",
				"cmd/run.go", "cmd/sub/deep.go",
				"export/export.go",
			},
		},
		{
			name: "exclude a subpackage of an included package",
			view: View{Include: []string{"cmd"}, Exclude: []string{"cmd/sub"}},
			want: []string{"cmd/run.go"},
		},
		{
			name: "glob exclude",
			view: View{Include: []string{"cmd"}, Exclude: []string{"**/sub"}},
			want: []string{"cmd/run.go"},
		},
		{
			name: "exclude that matches nothing changes nothing",
			view: View{Include: []string{"internal"}, Exclude: []string{"nothing"}},
			want: []string{"internal/app.go", "internal/app_test.go"},
		},
		{
			name: "exclude wins over include for the same package",
			view: View{Include: []string{"cmd"}, Exclude: []string{"cmd"}},
			want: nil,
		},
		{
			name: "exclude without include yields nothing",
			view: View{Exclude: []string{"cmd"}},
			want: nil,
		},
		{
			name: "empty view",
			view: View{},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exts := tt.exts
			if exts == nil {
				exts = goExts
			}
			got, err := GetFiles(&tt.view, exts, root)
			if err != nil {
				t.Fatalf("GetFiles returned error: %v", err)
			}
			if !slices.Equal(slashSorted(got), slashSorted(tt.want)) {
				t.Errorf("GetFiles = %v, want %v", slashSorted(got), slashSorted(tt.want))
			}
		})
	}
}

func TestGetFiles_ResultIsSortedWithForwardSlashes(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, "z.go", "b/b.go", "a/sub/a.go", "a/a.go")

	got, err := GetFiles(&View{Include: []string{"."}}, []string{"*.go"}, root)
	if err != nil {
		t.Fatalf("GetFiles returned error: %v", err)
	}

	want := []string{"a/a.go", "a/sub/a.go", "b/b.go", "z.go"}
	if !slices.Equal(got, want) {
		t.Errorf("GetFiles = %v, want %v (sorted, forward slashes, no slashSorted)", got, want)
	}
}

func TestGetFiles_NoExtensions(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, "main.go")

	for _, exts := range [][]string{nil, {}} {
		got, err := GetFiles(&View{Include: []string{"."}}, exts, root)
		if err == nil {
			t.Errorf("exts = %v: expected error, got files %v", exts, got)
		}
		if len(got) != 0 {
			t.Errorf("exts = %v: got %v, want empty result on error", exts, got)
		}
	}
}

func TestGetFiles_RestoresWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, "main.go")

	before, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	if _, err := GetFiles(&View{Include: []string{"."}}, []string{"*.go"}, root); err != nil {
		t.Fatalf("GetFiles returned error: %v", err)
	}

	after, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if after != before {
		t.Errorf("working directory = %q, want %q", after, before)
	}
}

func TestGetFiles_MalformedPattern(t *testing.T) {
	got, err := GetFiles(&View{Include: []string{"["}}, []string{"*.go"}, t.TempDir())
	if err == nil {
		t.Fatal("expected error for malformed pattern")
	}
	if len(got) != 0 {
		t.Errorf("got %v, want empty result on error", got)
	}
}

// Integration: a view loaded from disk must never pick up the cache folder.
func TestGetFiles_ExcludesCacheDirAfterLoad(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, "main.py", "pkg/mod.py", ".archlens/cache.json")
	anyFile := []string{"*"}

	// Control: without the injected exclusion, the cache file is picked up
	plain := View{Include: []string{"."}}
	got, err := GetFiles(&plain, anyFile, root)
	if err != nil {
		t.Fatalf("GetFiles returned error: %v", err)
	}
	if !slices.Contains(slashSorted(got), ".archlens/cache.json") {
		t.Fatalf("control failed: expected .archlens/cache.json in %v", slashSorted(got))
	}

	// Through Load, the cache exclusion is added automatically
	in, err := Load(writeConfig(t, `{"files": ["*"], "views": {"all": {"include": ["."]}}}`))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	view := in.Views["all"]
	got, err = GetFiles(&view, in.Files, root)
	if err != nil {
		t.Fatalf("GetFiles returned error: %v", err)
	}

	want := []string{"main.py", "pkg/mod.py"}
	if !slices.Equal(slashSorted(got), want) {
		t.Errorf("GetFiles = %v, want %v", slashSorted(got), want)
	}
}