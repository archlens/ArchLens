package gitutils

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// NOTE: these tests set environment variables (t.Setenv), so none of them may
// use t.Parallel(). Tests that need the git binary skip themselves without it.

var gitMissing bool

func TestMain(m *testing.M) {
	_, err := exec.LookPath("git")
	gitMissing = err != nil

	// Hermetic git: ignore user/system config and any inherited repository
	for _, k := range []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_COMMON_DIR"} {
		_ = os.Unsetenv(k)
	}
	_ = os.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	_ = os.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	_ = os.Setenv("GIT_AUTHOR_NAME", "ArchLens Test")
	_ = os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	_ = os.Setenv("GIT_COMMITTER_NAME", "ArchLens Test")
	_ = os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")

	os.Exit(m.Run())
}

// ---------- helpers ----------

func requireGit(t *testing.T) {
	t.Helper()
	if gitMissing {
		t.Skip("git binary not found in PATH")
	}
}

// git runs a git command in dir and returns its trimmed stdout.
func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
	if err != nil {
		var exitErr *exec.ExitError
		var stderr []byte
		if errors.As(err, &exitErr) {
			stderr = exitErr.Stderr
		}
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, stderr)
	}
	return strings.TrimSpace(string(out))
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// resolve follows symlinks (on macOS t.TempDir() is under /var, a symlink to /private/var).
func resolve(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", p, err)
	}
	return r
}

// initRepo creates an empty repository whose branch is "main".
func initRepo(t *testing.T) string {
	t.Helper()
	requireGit(t)
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	git(t, dir, "symbolic-ref", "HEAD", "refs/heads/main")
	return dir
}

// twoCommitRepo returns a repo where a.txt is "one" at tag v1 and "two" at HEAD.
func twoCommitRepo(t *testing.T) string {
	t.Helper()
	dir := initRepo(t)
	for i, content := range []string{"one", "two"} {
		writeFile(t, filepath.Join(dir, "a.txt"), content)
		git(t, dir, "add", "a.txt")
		git(t, dir, "commit", "-q", "-m", "update")
		if i == 0 {
			git(t, dir, "tag", "v1")
		}
	}
	return dir
}

// notARepo returns a directory that is guaranteed not to be inside a repository.
func notARepo(t *testing.T) string {
	t.Helper()
	requireGit(t)
	dir := t.TempDir()
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(resolve(t, dir)))
	return dir
}

// isolateTempDir points os.TempDir() at a fresh directory so leftover
// worktree directories can be detected.
func isolateTempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)
	t.Setenv("TMP", dir)
	t.Setenv("TEMP", dir)
	return dir
}

func leftoverWorktrees(t *testing.T, tmp string) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(tmp, "archlens-worktree-*"))
	if err != nil {
		t.Fatal(err)
	}
	return matches
}

// ---------- Prefix ----------

func Test_Prefix(t *testing.T) {
	repo := initRepo(t)
	if err := os.MkdirAll(filepath.Join(repo, "sub", "dir"), 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct{ name, rel, want string }{
		{"top level is empty", ".", ""},
		{"nested", "sub/dir", "sub/dir/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Prefix(filepath.Join(repo, filepath.FromSlash(tt.rel)))
			if err != nil {
				t.Fatalf("Prefix returned error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Prefix = %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_Prefix_NotARepo(t *testing.T) {
	got, err := Prefix(notARepo(t))
	if err == nil {
		t.Fatal("expected an error outside a repository")
	}
	if got != "" {
		t.Errorf("Prefix = %q, want empty on error", got)
	}
}

// ---------- AddWorktree ----------

func Test_AddWorktree_ChecksOutRef(t *testing.T) {
	isolateTempDir(t)
	dir := twoCommitRepo(t)

	tests := []struct{ ref, want string }{
		{"HEAD", "two"},
		{"HEAD~1", "one"},
		{"v1", "one"},
		{"main", "two"}, // checked out in the main worktree too
	}
	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			path, cleanup, err := AddWorktree(dir, tt.ref)
			if err != nil {
				t.Fatalf("AddWorktree returned error: %v", err)
			}
			t.Cleanup(cleanup)

			if got := readFile(t, filepath.Join(path, "a.txt")); got != tt.want {
				t.Errorf("a.txt = %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_AddWorktree_DetachedAndCleanedUp(t *testing.T) {
	tmp := isolateTempDir(t)
	dir := twoCommitRepo(t)

	// Uncommitted state in the caller's repo must survive
	writeFile(t, filepath.Join(dir, "a.txt"), "dirty")
	statusBefore := git(t, dir, "status", "--porcelain")
	branchesBefore := git(t, dir, "branch", "--list")

	path, cleanup, err := AddWorktree(dir, "v1")
	if err != nil {
		t.Fatalf("AddWorktree returned error: %v", err)
	}
	if got := git(t, path, "rev-parse", "--abbrev-ref", "HEAD"); got != "HEAD" {
		t.Errorf("worktree HEAD = %q, want a detached HEAD", got)
	}

	cleanup()

	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("worktree directory should be gone, Stat error: %v", err)
	}
	if left := leftoverWorktrees(t, tmp); len(left) != 0 {
		t.Errorf("leftover worktree dirs: %v", left)
	}
	if list := git(t, dir, "worktree", "list", "--porcelain"); strings.Contains(list, "archlens-worktree-") {
		t.Errorf("worktree should be unregistered:\n%s", list)
	}
	if got := git(t, dir, "status", "--porcelain"); got != statusBefore {
		t.Errorf("status changed:\nbefore: %s\nafter:  %s", statusBefore, got)
	}
	if got := git(t, dir, "branch", "--list"); got != branchesBefore {
		t.Errorf("branches changed:\nbefore: %s\nafter:  %s", branchesBefore, got)
	}

	cleanup() // calling it again must be harmless
}

func Test_AddWorktree_Errors(t *testing.T) {
	tmp := isolateTempDir(t)
	dir := twoCommitRepo(t)

	tests := []struct {
		name, dir, ref string
	}{
		{"unknown ref", dir, "no-such-ref"},
		{"ref that looks like a flag", dir, "--force"},
		{"not a repo", notARepo(t), "HEAD"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, cleanup, err := AddWorktree(tt.dir, tt.ref)
			if err == nil {
				if cleanup != nil {
					cleanup()
				}
				t.Fatal("expected an error")
			}
			if path != "" || cleanup != nil {
				t.Errorf("path = %q, cleanup nil = %v; want empty path and nil cleanup", path, cleanup == nil)
			}
			if left := leftoverWorktrees(t, tmp); len(left) != 0 {
				t.Errorf("temp dir should be removed on failure, found: %v", left)
			}
		})
	}
}

// ---------- EnsureCacheIgnored ----------

func Test_EnsureCacheIgnored_CreatesAndAppends(t *testing.T) {
	tests := []struct {
		name     string
		existing *string // nil = no .gitignore yet
		want     string
	}{
		{"missing file", nil, "**/.archlens/\n"},
		{"newline-terminated file", ptr("node_modules/\n"), "node_modules/\n**/.archlens/\n"},
		{"unterminated last line", ptr("node_modules/"), "node_modules/\n**/.archlens/\n"},
		{"comment and lookalikes do not count", ptr("# .archlens/\n.archlens-backup/\n"), "# .archlens/\n.archlens-backup/\n**/.archlens/\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, ".gitignore")
			if tt.existing != nil {
				writeFile(t, path, *tt.existing)
			}

			changed, err := EnsureCacheIgnored(root)
			if err != nil {
				t.Fatalf("EnsureCacheIgnored returned error: %v", err)
			}
			if !changed {
				t.Error("changed = false, want true")
			}
			if got := readFile(t, path); got != tt.want {
				t.Errorf(".gitignore = %q, want %q", got, tt.want)
			}
		})
	}
}

func ptr(s string) *string { return &s }

func Test_EnsureCacheIgnored_LeavesEquivalentEntriesAlone(t *testing.T) {
	tests := []struct{ name, content string }{
		{"plain", ".archlens\n"},
		{"anchored with slash", "/.archlens/\n"},
		{"any depth", "**/.archlens/\n"},
		{"glob contents", ".archlens/**\n"},
		{"surrounding whitespace", "  .archlens/  \n"},
		{"CRLF line endings", "node_modules/\r\n.archlens/\r\n"},
		{"unterminated last line", "node_modules/\n.archlens"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, ".gitignore")
			writeFile(t, path, tt.content)

			changed, err := EnsureCacheIgnored(root)
			if err != nil {
				t.Fatalf("EnsureCacheIgnored returned error: %v", err)
			}
			if changed {
				t.Error("changed = true, want false")
			}
			if got := readFile(t, path); got != tt.content {
				t.Errorf(".gitignore was modified: %q", got)
			}
		})
	}
}

func Test_EnsureCacheIgnored_IsIdempotent(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".gitignore"), "node_modules/")

	for i, want := range []bool{true, false} {
		changed, err := EnsureCacheIgnored(root)
		if err != nil {
			t.Fatalf("call %d returned error: %v", i+1, err)
		}
		if changed != want {
			t.Errorf("call %d: changed = %v, want %v", i+1, changed, want)
		}
	}
	if n := strings.Count(readFile(t, filepath.Join(root, ".gitignore")), ".archlens"); n != 1 {
		t.Errorf("expected exactly one .archlens entry, found %d", n)
	}
}

func Test_EnsureCacheIgnored_MissingProjectRoot(t *testing.T) {
	changed, err := EnsureCacheIgnored(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("expected an error when the project root does not exist")
	}
	if changed {
		t.Error("changed = true, want false on error")
	}
}

// Integration: git itself must treat the cache folder as ignored at any depth.
func Test_EnsureCacheIgnored_GitIgnoresCacheAtAnyDepth(t *testing.T) {
	repo := initRepo(t)
	writeFile(t, filepath.Join(repo, ".archlens", "cache.json"), "{}")
	writeFile(t, filepath.Join(repo, "sub", "deep", ".archlens", "cache.json"), "{}")
	writeFile(t, filepath.Join(repo, "src", "main.go"), "package main")

	if _, err := EnsureCacheIgnored(repo); err != nil {
		t.Fatalf("EnsureCacheIgnored returned error: %v", err)
	}

	status := git(t, repo, "status", "--porcelain", "--untracked-files=all")
	if strings.Contains(status, ".archlens") {
		t.Errorf("cache folders should be ignored at every depth:\n%s", status)
	}
	if !strings.Contains(status, "src/main.go") {
		t.Errorf("regular files must stay visible:\n%s", status)
	}
}

// ---------- FindRoot ----------

func Test_FindRoot(t *testing.T) {
	repo := initRepo(t)
	nested := filepath.Join(repo, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	for name, start := range map[string]string{"from the root": repo, "from a nested directory": nested} {
		t.Run(name, func(t *testing.T) {
			got, err := FindRoot(start)
			if err != nil {
				t.Fatalf("FindRoot returned error: %v", err)
			}
			if resolve(t, got) != resolve(t, repo) {
				t.Errorf("FindRoot = %q, want %q", got, repo)
			}
		})
	}
}

func Test_FindRoot_NotARepo(t *testing.T) {
	got, err := FindRoot(notARepo(t))
	if err == nil {
		t.Fatal("expected an error outside a repository")
	}
	if got != "" {
		t.Errorf("FindRoot = %q, want empty on error", got)
	}
}