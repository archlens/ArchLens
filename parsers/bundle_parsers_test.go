package parsers

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// ---------- Bundle* ----------

// bundleCase describes one Bundle<Language> function and what it must write.
type bundleCase struct {
	name   string             // language, used for subtest names
	bundle func(string) error // the function under test
	prefix string             // expected prefix of its errors
	files  map[string][]byte  // file name -> embedded content it must write
	squat  string             // one of the file names, used to block a write with a directory
}

func bundleCases() []bundleCase {
	return []bundleCase{
		{
			name:   "python",
			bundle: BundlePython,
			prefix: "bundle python parser",
			files:  map[string][]byte{"ast_parser.py": pythonParser},
			squat:  "ast_parser.py",
		},
		{
			name:   "go",
			bundle: BundleGo,
			prefix: "bundle go parser",
			files:  map[string][]byte{"ast_parser.go": goParser, "go.mod": goMod},
			squat:  "go.mod", // the second file, so the first one is already written
		},
		{
			name:   "java",
			bundle: BundleJava,
			prefix: "bundle java parser",
			files:  map[string][]byte{"AstParser.java": javaParser},
			squat:  "AstParser.java",
		},
	}
}

// dirEntries returns the names in dir, sorted.
func dirEntries(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir %s: %v", dir, err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names // os.ReadDir already sorts by file name
}

func sortedKeys(files map[string][]byte) []string {
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// assertBundleError checks that err is non-nil, wrapped, and carries the
// language-specific prefix.
func assertBundleError(t *testing.T, err error, prefix string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.HasPrefix(err.Error(), prefix+":") {
		t.Errorf("error %q should start with %q", err, prefix+":")
	}
	if errors.Unwrap(err) == nil {
		t.Errorf("error %q should wrap the underlying cause", err)
	}
}

// assertBundleContents checks that dir holds exactly the files in want, with matching content.
func assertBundleContents(t *testing.T, dir string, want map[string][]byte) {
	t.Helper()
	if got, wantNames := dirEntries(t, dir), sortedKeys(want); !slices.Equal(got, wantNames) {
		t.Errorf("files in %s = %v, want %v", dir, got, wantNames)
	}
	for name, wantData := range want {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Errorf("read %s: %v", name, err)
			continue
		}
		if string(got) != string(wantData) {
			t.Errorf("%s does not match the embedded parser (%d bytes written, %d embedded)", name, len(got), len(wantData))
		}
	}
}

func Test_Bundle_WritesExactlyTheExpectedFiles(t *testing.T) {
	for _, tc := range bundleCases() {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()

			if err := tc.bundle(dir); err != nil {
				t.Fatalf("bundle returned error: %v", err)
			}
			// The Go parser is stored as .txt in the repo; the written names must not carry that suffix
			assertBundleContents(t, dir, tc.files)
		})
	}
}

func Test_Bundle_CreatesMissingLocation(t *testing.T) {
	for _, tc := range bundleCases() {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "a", "b", "parsers")

			if err := tc.bundle(dir); err != nil {
				t.Fatalf("bundle returned error: %v", err)
			}
			assertBundleContents(t, dir, tc.files)
		})
	}
}

func Test_Bundle_OverwritesStaleCopiesAndLeavesOtherFilesAlone(t *testing.T) {
	for _, tc := range bundleCases() {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for name := range tc.files {
				writeTestFile(t, filepath.Join(dir, name), "stale content from an older version")
			}
			writeTestFile(t, filepath.Join(dir, "keep.txt"), "mine")

			if err := tc.bundle(dir); err != nil {
				t.Fatalf("bundle returned error: %v", err)
			}

			// Every bundled file matches the binary again, and the unrelated file is untouched
			want := map[string][]byte{"keep.txt": []byte("mine")}
			for name, data := range tc.files {
				want[name] = data
			}
			assertBundleContents(t, dir, want)
		})
	}
}

func Test_Bundle_IsIdempotent(t *testing.T) {
	for _, tc := range bundleCases() {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()

			for i := 1; i <= 2; i++ {
				if err := tc.bundle(dir); err != nil {
					t.Fatalf("call %d returned error: %v", i, err)
				}
			}
			assertBundleContents(t, dir, tc.files)
		})
	}
}

func Test_Bundle_LocationBlockedByAFile(t *testing.T) {
	kinds := map[string]func(blocker string) string{
		"location itself is a file":    func(blocker string) string { return blocker },
		"parent of location is a file": func(blocker string) string { return filepath.Join(blocker, "parsers") },
	}

	for _, tc := range bundleCases() {
		for kind, locationFor := range kinds {
			t.Run(tc.name+"/"+kind, func(t *testing.T) {
				blocker := filepath.Join(t.TempDir(), "blocker")
				writeTestFile(t, blocker, "x")

				assertBundleError(t, tc.bundle(locationFor(blocker)), tc.prefix)
			})
		}
	}
}

func Test_Bundle_TargetFileIsADirectory(t *testing.T) {
	for _, tc := range bundleCases() {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.Mkdir(filepath.Join(dir, tc.squat), 0o755); err != nil {
				t.Fatal(err)
			}

			assertBundleError(t, tc.bundle(dir), tc.prefix)
		})
	}
}

// Guards against embedding the wrong (or an empty) file.
func Test_Bundle_EmbeddedParsersLookRight(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		contains string
	}{
		{"python parser", pythonParser, "import"},
		{"go parser", goParser, "package main"},
		{"go.mod", goMod, "module "},
		{"java parser", javaParser, "class AstParser"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.data) == 0 {
				t.Fatal("embedded file is empty")
			}
			if !strings.Contains(string(tt.data), tt.contains) {
				t.Errorf("embedded file does not contain %q", tt.contains)
			}
		})
	}
}