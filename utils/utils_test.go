package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// NOTE: these tests change the working directory and swap os.Stdout, both of
// which are process-wide, so none of them may use t.Parallel().

// ---------- helpers ----------

// assertCwd checks the working directory after resolving symlinks (on macOS
// t.TempDir() lives under /var, which is a symlink to /private/var).
func assertCwd(t *testing.T, want string) {
	t.Helper()
	got, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	g, _ := filepath.EvalSymlinks(got)
	w, _ := filepath.EvalSymlinks(want)
	if g != w {
		t.Errorf("working directory = %q, want %q", g, w)
	}
}

// restoreWdAfter puts the working directory back even if a test fails halfway.
// Call it AFTER creating any t.TempDir()s so it runs first on cleanup.
func restoreWdAfter(t *testing.T) string {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
	return orig
}

// captureStdout redirects os.Stdout to a temp file and returns a func that
// reads what was written. Call it BEFORE NewPrettySugaredLogger, which binds
// to os.Stdout at construction.
func captureStdout(t *testing.T) func() string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatalf("create temp stdout: %v", err)
	}
	orig := os.Stdout
	os.Stdout = f
	t.Cleanup(func() {
		os.Stdout = orig
		_ = f.Close()
	})
	return func() string {
		data, err := os.ReadFile(f.Name())
		if err != nil {
			t.Fatalf("read captured stdout: %v", err)
		}
		return string(data)
	}
}

// ---------- WithWorkingDirectory ----------

func Test_WithWorkingDirectory_ChangesAndRestores(t *testing.T) {
	dir := t.TempDir()
	orig := restoreWdAfter(t)

	restore := WithWorkingDirectory(dir)
	assertCwd(t, dir)

	restore()
	assertCwd(t, orig)
}

func Test_WithWorkingDirectory_PanicsOnMissingDir(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	orig := restoreWdAfter(t)

	func() {
		defer func() {
			if recover() == nil {
				t.Error("expected a panic when the target directory does not exist")
			}
		}()
		WithWorkingDirectory(missing)
	}()

	// A failed change must leave the working directory untouched
	assertCwd(t, orig)
}

// ---------- NewPrettySugaredLogger ----------

func Test_NewPrettySugaredLogger_Levels(t *testing.T) {
	tests := []struct {
		name             string
		verbose, silence bool
		want             map[string]bool // message -> should be logged
	}{
		{"default logs info and above", false, false,
			map[string]bool{"debug-msg": false, "info-msg": true, "warn-msg": true, "error-msg": true}},
		{"verbose adds debug", true, false,
			map[string]bool{"debug-msg": true, "info-msg": true, "warn-msg": true, "error-msg": true}},
		{"silence keeps only errors", false, true,
			map[string]bool{"debug-msg": false, "info-msg": false, "warn-msg": false, "error-msg": true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			read := captureStdout(t)
			old := Sugar
			t.Cleanup(func() { Sugar = old })

			logger := NewPrettySugaredLogger(tt.verbose, tt.silence)
			if Sugar != logger {
				t.Error("package-level Sugar should be the logger that was returned")
			}

			logger.Debug("debug-msg")
			logger.Info("info-msg")
			logger.Warn("warn-msg")
			logger.Error("error-msg")
			_ = logger.Sync()

			out := read()
			for msg, want := range tt.want {
				if got := strings.Contains(out, msg); got != want {
					t.Errorf("%q logged = %v, want %v\noutput:\n%s", msg, got, want, out)
				}
			}
		})
	}
}