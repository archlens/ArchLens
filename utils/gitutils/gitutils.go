// Package gitutil holds small helpers around the git CLI.
package gitutils

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"errors"
)

const cachePattern = "**/.archlens/"

func run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return string(out), nil
}

// Prefix returns dir's path relative to the top of its repository, with forward
// slashes and a trailing slash ("" when dir is the top level).
func Prefix(dir string) (string, error) {
	out, err := run(dir, "rev-parse", "--show-prefix")
	return strings.TrimSpace(out), err
}

// AddWorktree checks ref out into a fresh temporary directory (detached, so the
// caller's working tree, index and branches are never touched) and returns that
// directory's path together with a cleanup function that removes it again.
func AddWorktree(dir, ref string) (path string, cleanup func(), err error) {
	if strings.HasPrefix(ref, "-") {
		return "", nil, fmt.Errorf("invalid git ref %q", ref)
	}
	tmp, err := os.MkdirTemp("", "archlens-worktree-*")
	if err != nil {
		return "", nil, err
	}
	if _, err := run(dir, "worktree", "add", "--detach", tmp, ref); err != nil {
		_ = os.RemoveAll(tmp)
		return "", nil, err
	}
	cleanup = func() {
		_, _ = run(dir, "worktree", "remove", "--force", tmp)
		_ = os.RemoveAll(tmp)
		_, _ = run(dir, "worktree", "prune")
	}
	return tmp, cleanup, nil
}

// equivalent entries that already ignore the cache folder
var cacheVariants = map[string]bool{
	".archlens":       true,
	".archlens/":      true,
	"/.archlens":      true,
	"/.archlens/":     true,
	"**/.archlens":    true,
	"**/.archlens/":   true,
	".archlens/*":     true,
	".archlens/**":    true,
	"**/.archlens/*":  true,
	"**/.archlens/**": true,
}

// EnsureCacheIgnored adds the ArchLens cache folder to the .gitignore in
// projectRoot, creating the file if needed. It reports whether it changed anything.
func EnsureCacheIgnored(projectRoot string) (bool, error) {
	path := filepath.Join(projectRoot, ".gitignore")

	// A missing file is fine: data stays empty and the file is created below
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("reading %s: %w", path, err)
	}

	for _, line := range strings.Split(string(data), "\n") {
		if cacheVariants[strings.TrimSpace(line)] {
			return false, nil // already ignored
		}
	}

	// Don't glue the entry onto an unterminated last line
	needsNewline := len(data) > 0 && data[len(data)-1] != '\n'

	out, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return false, fmt.Errorf("opening %s for writing: %w", path, err)
	}

	entry := cachePattern + "\n"
	if needsNewline {
		entry = "\n" + entry
	}
	if _, err := out.WriteString(entry); err != nil {
		_ = out.Close()
		return false, fmt.Errorf("writing %s: %w", path, err)
	}

	// Close can report a failed flush, so surface it
	if err := out.Close(); err != nil {
		return false, fmt.Errorf("closing %s: %w", path, err)
	}
	return true, nil
}

// FindRoot returns the root of the git working tree containing dir.
func FindRoot(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not inside a git repository (%s): %w", strings.TrimSpace(stderr.String()), err)
	}

	// Git prints forward slashes even on Windows
	return filepath.FromSlash(strings.TrimSpace(string(out))), nil
}
