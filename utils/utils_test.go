package utils

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// NOTE: these tests change the working directory and swap os.Stdout, both of
// which are process-wide, so none of them may use t.Parallel().

// ---------- helpers ----------

func mustGetwd(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	return wd
}

// assertSameDir compares directories after resolving symlinks (on macOS
// t.TempDir() lives under /var, which is a symlink to /private/var).
func assertSameDir(t *testing.T, got, want string) {
	t.Helper()
	g, err := filepath.EvalSymlinks(got)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", got, err)
	}
	w, err := filepath.EvalSymlinks(want)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", want, err)
	}
	if g != w {
		t.Errorf("working directory = %q, want %q", g, w)
	}
}

// restoreWdAfter makes sure the test leaves the working directory as it found
// it, even if an assertion fails halfway. Call it AFTER creating any
// t.TempDir()s so it runs first on cleanup (some OSes can't delete the cwd).
func restoreWdAfter(t *testing.T) {
	t.Helper()
	orig := mustGetwd(t)
	t.Cleanup(func() {
		if err := os.Chdir(orig); err != nil {
			t.Errorf("could not restore working directory: %v", err)
		}
	})
}

// panics reports whether f panicked.
func panics(f func()) (didPanic bool) {
	defer func() { didPanic = recover() != nil }()
	f()
	return false
}

// captureStdout redirects os.Stdout to a temp file for the rest of the test
// and returns a func that reads everything written so far. It must be called
// BEFORE NewPrettySugaredLogger, which binds to os.Stdout at construction.
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

// restoreSugar puts the package-level Sugar back after the test.
func restoreSugar(t *testing.T) {
	t.Helper()
	old := Sugar
	t.Cleanup(func() { Sugar = old })
}

// ---------- WithWorkingDirectory ----------

func Test_WithWorkingDirectory(t *testing.T) {
	current, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	restore := WithWorkingDirectory("..")

	after, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	if after == current {
		t.Errorf("Working directory didn't change\n Before: %s\n After: %s\n", current, after)
	}

	restore()

	after, err = os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	if after != current {
		t.Errorf("Working directory should be the same after restore\n Before: %s\n After: %s\n", current, after)
	}
}

func Test_WithWorkingDirectory_ChangesToGivenDir(t *testing.T) {
	dir := t.TempDir()
	restoreWdAfter(t)
	orig := mustGetwd(t)

	restore := WithWorkingDirectory(dir)
	assertSameDir(t, mustGetwd(t), dir)

	restore()
	assertSameDir(t, mustGetwd(t), orig)
}

func Test_WithWorkingDirectory_RelativePath(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	restoreWdAfter(t)
	orig := mustGetwd(t)

	restoreDir := WithWorkingDirectory(dir)
	restoreSub := WithWorkingDirectory("sub")
	assertSameDir(t, mustGetwd(t), sub)

	restoreSub()
	assertSameDir(t, mustGetwd(t), dir)

	restoreDir()
	assertSameDir(t, mustGetwd(t), orig)
}

func Test_WithWorkingDirectory_NestedRestoresInOrder(t *testing.T) {
	a := t.TempDir()
	b := t.TempDir()
	restoreWdAfter(t)
	orig := mustGetwd(t)

	restoreA := WithWorkingDirectory(a)
	restoreB := WithWorkingDirectory(b)
	assertSameDir(t, mustGetwd(t), b)

	restoreB()
	assertSameDir(t, mustGetwd(t), a)

	restoreA()
	assertSameDir(t, mustGetwd(t), orig)
}

func Test_WithWorkingDirectory_PanicsOnMissingDir(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	restoreWdAfter(t)
	orig := mustGetwd(t)

	if !panics(func() { WithWorkingDirectory(missing) }) {
		t.Error("expected a panic when the target directory does not exist")
	}

	// A failed change must leave the working directory untouched
	assertSameDir(t, mustGetwd(t), orig)
}

func Test_WithWorkingDirectory_RestorePanicsWhenOriginalIsGone(t *testing.T) {
	origDir := t.TempDir()
	target := t.TempDir()
	restoreWdAfter(t)

	if err := os.Chdir(origDir); err != nil {
		t.Fatal(err)
	}
	restore := WithWorkingDirectory(target)

	// The directory we would return to disappears while we are away
	if err := os.Remove(origDir); err != nil {
		t.Fatalf("remove original dir: %v", err)
	}

	if !panics(restore) {
		t.Error("expected restore to panic when the original directory no longer exists")
	}
}

// ---------- NewPrettySugaredLogger ----------

func Test_NewPrettySugaredLogger_SetsGlobalSugar(t *testing.T) {
	captureStdout(t)
	restoreSugar(t)
	Sugar = nil

	logger := NewPrettySugaredLogger(false, false)

	if logger == nil {
		t.Fatal("returned a nil logger")
	}
	if Sugar != logger {
		t.Error("package-level Sugar should be the logger that was returned")
	}
}

func Test_NewPrettySugaredLogger_Levels(t *testing.T) {
	tests := []struct {
		name                                       string
		verbose, silence                           bool
		wantDebug, wantInfo, wantWarn, wantError bool
	}{
		{
			name:      "default logs info and above",
			wantDebug: false, wantInfo: true, wantWarn: true, wantError: true,
		},
		{
			name:    "verbose adds debug",
			verbose: true,
			wantDebug: true, wantInfo: true, wantWarn: true, wantError: true,
		},
		{
			name:    "silence keeps only errors",
			silence: true,
			wantDebug: false, wantInfo: false, wantWarn: false, wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			read := captureStdout(t)
			restoreSugar(t)

			logger := NewPrettySugaredLogger(tt.verbose, tt.silence)
			logger.Debug("debug-msg")
			logger.Info("info-msg")
			logger.Warn("warn-msg")
			logger.Error("error-msg")
			_ = logger.Sync()

			out := read()
			check := func(msg string, want bool) {
				t.Helper()
				if got := strings.Contains(out, msg); got != want {
					t.Errorf("%q logged = %v, want %v\noutput:\n%s", msg, got, want, out)
				}
			}
			check("debug-msg", tt.wantDebug)
			check("info-msg", tt.wantInfo)
			check("warn-msg", tt.wantWarn)
			check("error-msg", tt.wantError)
		})
	}
}

func Test_NewPrettySugaredLogger_OutputFormat(t *testing.T) {
	read := captureStdout(t)
	restoreSugar(t)

	logger := NewPrettySugaredLogger(false, false)
	logger.Info("hello world")
	_ = logger.Sync()

	out := read()

	if !strings.Contains(out, "INFO") {
		t.Errorf("expected capitalized level INFO in output:\n%s", out)
	}
	if !strings.Contains(out, "\x1b[") {
		t.Errorf("expected ANSI color codes around the level:\n%s", out)
	}

	// ISO8601 timestamp, e.g. 2026-10-01T12:00:00.000+0200 (or Z for UTC)
	timestamp := regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}(Z|[+-]\d{4})`)
	if !timestamp.MatchString(out) {
		t.Errorf("expected an ISO8601 timestamp in output:\n%s", out)
	}

	// Short caller (dir/file.go:line) pointing at this test file, then the message
	caller := regexp.MustCompile(`\t[^/\t]+/[^/\t]+_test\.go:\d+\thello world`)
	if !caller.MatchString(out) {
		t.Errorf("expected a short caller followed by the message:\n%s", out)
	}

	// Console encoding, not JSON
	if strings.Contains(out, `"msg"`) || strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Errorf("expected console output, got JSON:\n%s", out)
	}
}

func Test_NewPrettySugaredLogger_StacktraceOnlyOnError(t *testing.T) {
	read := captureStdout(t)
	restoreSugar(t)

	logger := NewPrettySugaredLogger(false, false)

	logger.Warn("warn-msg")
	_ = logger.Sync()
	if out := read(); strings.Contains(out, "testing.tRunner") {
		t.Errorf("warn should not include a stacktrace:\n%s", out)
	}

	logger.Error("error-msg")
	_ = logger.Sync()
	if out := read(); !strings.Contains(out, "testing.tRunner") {
		t.Errorf("error should include a stacktrace:\n%s", out)
	}
}