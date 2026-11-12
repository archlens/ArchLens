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
	// (for example when the tests are started from a git hook).
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

// git runs a git command in dir and returns its trimmed stdout; it fails the test on error.
func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		var stderr []byte
		var exitErr *exec.ExitError
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
		t.Fatalf("mkdir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

// resolve follows symlinks (on macOS t.TempDir() lives under /var, a symlink
// to /private/var, while git reports the resolved path).
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

// commitFile writes a file, commits it and returns the new commit's SHA.
func commitFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	writeFile(t, filepath.Join(dir, filepath.FromSlash(name)), content)
	git(t, dir, "add", name)
	git(t, dir, "commit", "-q", "-m", "update "+name)
	return git(t, dir, "rev-parse", "HEAD")
}

// twoCommitRepo returns a repo where a.txt is "one" at the commit tagged v1
// and "two" at HEAD (main).
func twoCommitRepo(t *testing.T) (dir, first, second string) {
	t.Helper()
	dir = initRepo(t)
	first = commitFile(t, dir, "a.txt", "one")
	git(t, dir, "tag", "v1")
	second = commitFile(t, dir, "a.txt", "two")
	return dir, first, second
}

// notARepo returns a directory that is guaranteed not to be inside a repository.
func notARepo(t *testing.T) string {
	t.Helper()
	requireGit(t)
	dir := t.TempDir()
	// Stop git from discovering a repository in any parent of the temp dir
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
		t.Fatalf("glob: %v", err)
	}
	return matches
}

// ---------- Prefix ----------

func Test_Prefix(t *testing.T) {
	repo := initRepo(t)
	for _, d := range []string{"sub", "sub/dir", "my dir/sub dir"} {
		if err := os.MkdirAll(filepath.Join(repo, filepath.FromSlash(d)), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name string
		rel  string
		want string
	}{
		{"top level is empty", ".", ""},
		{"one level down", "sub", "sub/"},
		{"nested", "sub/dir", "sub/dir/"},
		{"paths with spaces", "my dir/sub dir", "my dir/sub dir/"},
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
	if !strings.Contains(err.Error(), "git rev-parse --show-prefix") {
		t.Errorf("error should name the failing git command, got: %v", err)
	}
}

func Test_Prefix_MissingDir(t *testing.T) {
	requireGit(t)
	if _, err := Prefix(filepath.Join(t.TempDir(), "does-not-exist")); err == nil {
		t.Fatal("expected an error for a missing directory")
	}
}

// ---------- AddWorktree ----------

func Test_AddWorktree_ChecksOutRef(t *testing.T) {
	tmp := isolateTempDir(t)
	dir, first, _ := twoCommitRepo(t)

	tests := []struct {
		name string
		ref  string
		want string
	}{
		{"HEAD", "HEAD", "two"},
		{"parent of HEAD", "HEAD~1", "one"},
		{"tag", "v1", "one"},
		{"full SHA", first, "one"},
		{"branch that is checked out in the main worktree", "main", "two"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, cleanup, err := AddWorktree(dir, tt.ref)
			if err != nil {
				t.Fatalf("AddWorktree returned error: %v", err)
			}
			t.Cleanup(cleanup)

			if got := readFile(t, filepath.Join(path, "a.txt")); got != tt.want {
				t.Errorf("a.txt = %q, want %q", got, tt.want)
			}
			// Sanity check for the leftover-detection used in other tests
			if filepath.Dir(path) != filepath.Clean(tmp) || !strings.HasPrefix(filepath.Base(path), "archlens-worktree-") {
				t.Errorf("worktree %q should be a archlens-worktree-* dir inside %q", path, tmp)
			}
		})
	}
}

func Test_AddWorktree_IsDetachedAndLeavesCallerUntouched(t *testing.T) {
	isolateTempDir(t)
	dir, _, _ := twoCommitRepo(t)

	// Give the caller's repository some uncommitted state
	writeFile(t, filepath.Join(dir, "a.txt"), "dirty")
	writeFile(t, filepath.Join(dir, "scratch.txt"), "untracked")

	headBefore := git(t, dir, "rev-parse", "HEAD")
	branchesBefore := git(t, dir, "branch", "--list")
	statusBefore := git(t, dir, "status", "--porcelain")

	path, cleanup, err := AddWorktree(dir, "v1")
	if err != nil {
		t.Fatalf("AddWorktree returned error: %v", err)
	}

	// Detached: HEAD is not on a branch
	if got := git(t, path, "rev-parse", "--abbrev-ref", "HEAD"); got != "HEAD" {
		t.Errorf("worktree HEAD = %q, want a detached HEAD", got)
	}
	// Lives outside the caller's repository
	if strings.HasPrefix(resolve(t, path), resolve(t, dir)) {
		t.Errorf("worktree %q should not be inside the repository %q", path, dir)
	}

	cleanup()

	if got := git(t, dir, "rev-parse", "HEAD"); got != headBefore {
		t.Errorf("HEAD moved: %s -> %s", headBefore, got)
	}
	if got := git(t, dir, "branch", "--list"); got != branchesBefore {
		t.Errorf("branches changed:\nbefore: %s\nafter:  %s", branchesBefore, got)
	}
	if got := git(t, dir, "status", "--porcelain"); got != statusBefore {
		t.Errorf("status changed:\nbefore: %s\nafter:  %s", statusBefore, got)
	}
	if got := readFile(t, filepath.Join(dir, "a.txt")); got != "dirty" {
		t.Errorf("caller's a.txt = %q, want it untouched", got)
	}
}

func Test_AddWorktree_CleanupRemovesWorktree(t *testing.T) {
	tmp := isolateTempDir(t)
	dir, _, _ := twoCommitRepo(t)

	path, cleanup, err := AddWorktree(dir, "HEAD")
	if err != nil {
		t.Fatalf("AddWorktree returned error: %v", err)
	}

	registered := filepath.ToSlash(resolve(t, path))
	if list := git(t, dir, "worktree", "list", "--porcelain"); !strings.Contains(list, registered) {
		t.Fatalf("worktree %q not registered:\n%s", registered, list)
	}

	cleanup()

	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("worktree directory should be gone, Stat error: %v", err)
	}
	if list := git(t, dir, "worktree", "list", "--porcelain"); strings.Contains(list, registered) {
		t.Errorf("worktree should be unregistered:\n%s", list)
	}
	if left := leftoverWorktrees(t, tmp); len(left) != 0 {
		t.Errorf("leftover worktree dirs: %v", left)
	}

	// Calling cleanup again must be harmless
	cleanup()
}

func Test_AddWorktree_CleanupAfterManualDeletion(t *testing.T) {
	isolateTempDir(t)
	dir, _, _ := twoCommitRepo(t)

	path, cleanup, err := AddWorktree(dir, "HEAD")
	if err != nil {
		t.Fatalf("AddWorktree returned error: %v", err)
	}
	if err := os.RemoveAll(path); err != nil {
		t.Fatal(err)
	}

	cleanup() // must not panic, and must prune the stale registration

	registered := filepath.ToSlash(path)
	if list := git(t, dir, "worktree", "list", "--porcelain"); strings.Contains(list, filepath.Base(registered)) {
		t.Errorf("stale worktree entry should be pruned:\n%s", list)
	}
}

func Test_AddWorktree_RejectsRefsThatLookLikeFlags(t *testing.T) {
	tmp := isolateTempDir(t)
	dir, _, _ := twoCommitRepo(t)

	for _, ref := range []string{"-b", "--force", "-", "--detach"} {
		t.Run(ref, func(t *testing.T) {
			path, cleanup, err := AddWorktree(dir, ref)
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
				t.Errorf("leftover worktree dirs: %v", left)
			}
		})
	}
}

func Test_AddWorktree_UnknownRef(t *testing.T) {
	tmp := isolateTempDir(t)
	dir, _, _ := twoCommitRepo(t)

	path, cleanup, err := AddWorktree(dir, "no-such-ref")
	if err == nil {
		cleanup()
		t.Fatal("expected an error for an unknown ref")
	}
	if path != "" || cleanup != nil {
		t.Errorf("path = %q, cleanup nil = %v; want empty path and nil cleanup", path, cleanup == nil)
	}
	if !strings.Contains(err.Error(), "worktree add") {
		t.Errorf("error should name the failing git command, got: %v", err)
	}
	if left := leftoverWorktrees(t, tmp); len(left) != 0 {
		t.Errorf("temp dir should be removed on failure, found: %v", left)
	}
}

func Test_AddWorktree_NotARepo(t *testing.T) {
	tmp := isolateTempDir(t)

	path, cleanup, err := AddWorktree(notARepo(t), "HEAD")
	if err == nil {
		cleanup()
		t.Fatal("expected an error outside a repository")
	}
	if path != "" || cleanup != nil {
		t.Errorf("path = %q, cleanup nil = %v; want empty path and nil cleanup", path, cleanup == nil)
	}
	if left := leftoverWorktrees(t, tmp); len(left) != 0 {
		t.Errorf("temp dir should be removed on failure, found: %v", left)
	}
}

// ---------- EnsureCacheIgnored ----------

func Test_EnsureCacheIgnored_CreatesMissingFile(t *testing.T) {
	root := t.TempDir()

	changed, err := EnsureCacheIgnored(root)
	if err != nil {
		t.Fatalf("EnsureCacheIgnored returned error: %v", err)
	}
	if !changed {
		t.Error("changed = false, want true")
	}
	if got := readFile(t, filepath.Join(root, ".gitignore")); got != "**/.archlens/\n" {
		t.Errorf(".gitignore = %q, want %q", got, "**/.archlens/\n")
	}
}

func Test_EnsureCacheIgnored_Appends(t *testing.T) {
	tests := []struct {
		name     string
		existing string
		want     string
	}{
		{"empty file", "", "**/.archlens/\n"},
		{"newline-terminated file", "node_modules/\n", "node_modules/\n**/.archlens/\n"},
		{"unterminated last line", "node_modules/", "node_modules/\n**/.archlens/\n"},
		{"commented mention does not count", "# .archlens/\n", "# .archlens/\n**/.archlens/\n"},
		{
			"lookalike entries do not count",
			".archlens-backup/\narchlens/\n",
			".archlens-backup/\narchlens/\n**/.archlens/\n",
		},
		{"negation does not count", "!.archlens/\n", "!.archlens/\n**/.archlens/\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, ".gitignore")
			writeFile(t, path, tt.existing)

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

func Test_EnsureCacheIgnored_LeavesEquivalentEntriesAlone(t *testing.T) {
	variants := []string{
		".archlens", ".archlens/", "/.archlens", "/.archlens/",
		"**/.archlens", "**/.archlens/",
		".archlens/*", ".archlens/**", "**/.archlens/*", "**/.archlens/**",
	}

	for _, v := range variants {
		t.Run(v, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, ".gitignore")
			content := "node_modules/\n" + v + "\ndist/\n"
			writeFile(t, path, content)

			changed, err := EnsureCacheIgnored(root)
			if err != nil {
				t.Fatalf("EnsureCacheIgnored returned error: %v", err)
			}
			if changed {
				t.Error("changed = true, want false")
			}
			if got := readFile(t, path); got != content {
				t.Errorf(".gitignore was modified: %q", got)
			}
		})
	}
}

func Test_EnsureCacheIgnored_ToleratesWhitespaceAndLineEndings(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"surrounding spaces", "  .archlens/  \n"},
		{"tabs", "\t**/.archlens/\t\n"},
		{"CRLF line endings", "node_modules/\r\n.archlens/\r\n"},
		{"entry on unterminated last line", "node_modules/\n.archlens"},
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

	for i, want := range []bool{true, false, false} {
		changed, err := EnsureCacheIgnored(root)
		if err != nil {
			t.Fatalf("call %d returned error: %v", i+1, err)
		}
		if changed != want {
			t.Errorf("call %d: changed = %v, want %v", i+1, changed, want)
		}
	}

	got := readFile(t, filepath.Join(root, ".gitignore"))
	if n := strings.Count(got, ".archlens"); n != 1 {
		t.Errorf("expected exactly one .archlens entry, found %d in %q", n, got)
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

func Test_EnsureCacheIgnored_GitignoreIsADirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".gitignore"), 0o755); err != nil {
		t.Fatal(err)
	}

	changed, err := EnsureCacheIgnored(root)
	if err == nil {
		t.Fatal("expected an error when .gitignore is a directory")
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

	status := func() string {
		return git(t, repo, "status", "--porcelain", "--untracked-files=all")
	}

	// Control: before the entry exists, the cache files show up as untracked
	before := status()
	if !strings.Contains(before, ".archlens/cache.json") || !strings.Contains(before, "sub/deep/.archlens/cache.json") {
		t.Fatalf("control failed, cache files should be untracked:\n%s", before)
	}

	if _, err := EnsureCacheIgnored(repo); err != nil {
		t.Fatalf("EnsureCacheIgnored returned error: %v", err)
	}

	after := status()
	if strings.Contains(after, ".archlens") {
		t.Errorf("cache folders should be ignored at every depth:\n%s", after)
	}
	if !strings.Contains(after, "src/main.go") {
		t.Errorf("regular files must stay visible:\n%s", after)
	}
}

// ---------- FindRoot ----------

func Test_FindRoot(t *testing.T) {
	repo := initRepo(t)
	nested := filepath.Join(repo, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		start string
	}{
		{"from the root", repo},
		{"from a nested directory", nested},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FindRoot(tt.start)
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
	if !strings.Contains(err.Error(), "not inside a git repository") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func Test_FindRoot_MissingDir(t *testing.T) {
	requireGit(t)
	if _, err := FindRoot(filepath.Join(t.TempDir(), "does-not-exist")); err == nil {
		t.Fatal("expected an error for a missing directory")
	}
}

func Test_FindRoot_InLinkedWorktree(t *testing.T) {
	isolateTempDir(t)
	dir, _, _ := twoCommitRepo(t)

	path, cleanup, err := AddWorktree(dir, "HEAD")
	if err != nil {
		t.Fatalf("AddWorktree returned error: %v", err)
	}
	t.Cleanup(cleanup)

	got, err := FindRoot(path)
	if err != nil {
		t.Fatalf("FindRoot returned error: %v", err)
	}
	if resolve(t, got) != resolve(t, path) {
		t.Errorf("FindRoot = %q, want the worktree's own root %q", got, path)
	}
}