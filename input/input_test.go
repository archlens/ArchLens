package input

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/archlens/ArchLens/utils"
	"go.uber.org/zap"
)

// NOTE: these tests change the working directory (GetFiles does it itself and
// matchFiles needs it), so none of them may use t.Parallel().

func TestMain(m *testing.M) {
	// logger is captured from utils.Sugar at package init; make sure it is usable
	logger = zap.NewNop().Sugar()
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

// ---------- Load ----------

func TestLoad_ParsesAllFields(t *testing.T) {
	path := writeConfig(t, `{
		"name": "demo",
		"github": {"url": "https://github.com/archlens/demo", "branch": "main"},
		"rootFolder": "src",
		"saveLocation": "docs/arch",
		"runCommand": "python3 ast_parser.py",
		"views": {
			"complete": {"include": ["**/*.py"], "exclude": ["venv/**"]}
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
	if !slices.Equal(view.Include, []string{"**/*.py"}) {
		t.Errorf("Include = %v", view.Include)
	}
	if !slices.Equal(view.Exclude, []string{"venv/**", ".archlens/*"}) {
		t.Errorf("Exclude = %v", view.Exclude)
	}
}

func TestLoad_AppendsCacheExclusionToEveryView(t *testing.T) {
	path := writeConfig(t, `{"views": {
		"noExclude":    {"include": ["*.go"]},
		"emptyExclude": {"include": ["*.go"], "exclude": []},
		"withExclude":  {"include": ["*.go"], "exclude": ["vendor/**", "*_test.go"]}
	}}`)

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	want := map[string][]string{
		"noExclude":    {".archlens/*"},
		"emptyExclude": {".archlens/*"},
		"withExclude":  {"vendor/**", "*_test.go", ".archlens/*"},
	}
	if len(got.Views) != len(want) {
		t.Fatalf("got %d views, want %d", len(got.Views), len(want))
	}
	for name, wantExclude := range want {
		view, ok := got.Views[name]
		if !ok {
			t.Errorf("view %q missing", name)
			continue
		}
		if !slices.Equal(view.Exclude, wantExclude) {
			t.Errorf("view %q Exclude = %v, want %v", name, view.Exclude, wantExclude)
		}
		if !slices.Equal(view.Include, []string{"*.go"}) {
			t.Errorf("view %q Include = %v, want [*.go]", name, view.Include)
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
	)

	allGo := []string{
		"main.go", "main_test.go",
		"internal/app.go", "internal/app_test.go",
		"vendor/lib/lib.go",
	}

	tests := []struct {
		name string
		view View
		want []string
	}{
		{
			name: "include only",
			view: View{Include: []string{"**/*.go"}},
			want: allGo,
		},
		{
			name: "exclude removes matches",
			view: View{
				Include: []string{"**/*.go"},
				Exclude: []string{"vendor/**", "**/*_test.go"},
			},
			want: []string{"main.go", "internal/app.go"},
		},
		{
			name: "exclude that matches nothing changes nothing",
			view: View{Include: []string{"**/*.go"}, Exclude: []string{"*.rs"}},
			want: allGo,
		},
		{
			name: "exclude wins over include for the same file",
			view: View{Include: []string{"main.go"}, Exclude: []string{"main.go"}},
			want: nil,
		},
		{
			name: "exclude without include yields nothing",
			view: View{Exclude: []string{"**/*.go"}},
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
			got, err := GetFiles(&tt.view, root)
			if err != nil {
				t.Fatalf("GetFiles returned error: %v", err)
			}
			if !slices.Equal(slashSorted(got), slashSorted(tt.want)) {
				t.Errorf("GetFiles = %v, want %v", slashSorted(got), slashSorted(tt.want))
			}
		})
	}
}

func TestGetFiles_RestoresWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, "main.go")

	before, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	if _, err := GetFiles(&View{Include: []string{"*.go"}}, root); err != nil {
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
	got, err := GetFiles(&View{Include: []string{"["}}, t.TempDir())
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

	// Control: without the injected exclusion, the cache file is picked up
	plain := View{Include: []string{"**/*"}}
	got, err := GetFiles(&plain, root)
	if err != nil {
		t.Fatalf("GetFiles returned error: %v", err)
	}
	if !slices.Contains(slashSorted(got), ".archlens/cache.json") {
		t.Fatalf("control failed: expected .archlens/cache.json in %v", slashSorted(got))
	}

	// Through Load, the cache exclusion is added automatically
	in, err := Load(writeConfig(t, `{"views": {"all": {"include": ["**/*"]}}}`))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	view := in.Views["all"]
	got, err = GetFiles(&view, root)
	if err != nil {
		t.Fatalf("GetFiles returned error: %v", err)
	}

	want := []string{"main.py", "pkg/mod.py"}
	if !slices.Equal(slashSorted(got), want) {
		t.Errorf("GetFiles = %v, want %v", slashSorted(got), want)
	}
}