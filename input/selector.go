package input

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// Selector decides which package nodes a view shows and which files it needs.
type Selector struct {
	include, exclude, files []string
}

// Selectors builds one Selector per view, with the view's own files patterns
// or the top-level ones.
func (in *Input) Selectors() (map[string]*Selector, error) {
	out := make(map[string]*Selector, len(in.Views))
	for name, v := range in.Views {
		files := in.Files
		if len(v.Files) > 0 {
			files = v.Files
		}
		if len(files) == 0 {
			return nil, fmt.Errorf(`view %q: no file patterns, set "files" (e.g. ["*.py"]) at the top level or in the view`, name)
		}
		s, err := newSelector(v, files)
		if err != nil {
			return nil, fmt.Errorf("view %q: %w", name, err)
		}
		out[name] = s
	}
	return out, nil
}

func newSelector(v View, files []string) (*Selector, error) {
	inc, err := cleanPatterns("include", v.Include)
	if err != nil {
		return nil, err
	}
	exc, err := cleanPatterns("exclude", v.Exclude)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		if !doublestar.ValidatePattern(strings.TrimPrefix(f, "!")) {
			return nil, fmt.Errorf("invalid files pattern %q", f)
		}
	}
	return &Selector{include: inc, exclude: exc, files: files}, nil
}

// cleanPatterns normalizes "/core/model/" and "./core/model" to "core/model".
func cleanPatterns(kind string, patterns []string) ([]string, error) {
	out := make([]string, 0, len(patterns))
	for _, p := range patterns {
		c := path.Clean(strings.Trim(filepath.ToSlash(p), "/"))
		if !doublestar.ValidatePattern(c) {
			return nil, fmt.Errorf("invalid %s pattern %q", kind, p)
		}
		out = append(out, c)
	}
	return out, nil
}

// prefixes lists a package's directory prefixes, shortest first:
// "a/b/c" gives a, a/b, a/b/c. The root package "." has only itself.
func prefixes(pkg string) []string {
	if pkg == "." || pkg == "" {
		return []string{"."}
	}
	parts := strings.Split(pkg, "/")
	out := make([]string, len(parts))
	for i := range parts {
		out[i] = strings.Join(parts[:i+1], "/")
	}
	return out
}

func matchesAny(patterns []string, name string) bool {
	for _, p := range patterns {
		if ok, _ := doublestar.Match(p, name); ok {
			return true
		}
	}
	return false
}

// Excluded reports whether pkg, or one of its parent directories, matches an
// exclude pattern.
func (s *Selector) Excluded(pkg string) bool {
	for _, p := range prefixes(pkg) {
		if matchesAny(s.exclude, p) {
			return true
		}
	}
	return false
}

// Node returns the node pkg is shown as. An excluded package is dropped. With
// no include list every package is its own node. Otherwise the deepest
// ancestor of pkg (pkg itself included) that matches an include pattern is the
// node, and a package with no match is dropped. "." only matches the root
// package, not everything.
func (s *Selector) Node(pkg string) (string, bool) {
	if s.Excluded(pkg) {
		return "", false
	}
	if len(s.include) == 0 {
		return pkg, true
	}
	ps := prefixes(pkg)
	for i := len(ps) - 1; i >= 0; i-- {
		if matchesAny(s.include, ps[i]) {
			return ps[i], true
		}
	}
	return "", false
}

// MatchFile reports whether a file (relative to the root, "/"-separated) is
// one of the view's source files. Patterns without a "/" match the base name
// at any depth ("*.py"); the last matching pattern wins and "!" negates.
func (s *Selector) MatchFile(rel string) bool {
	matched := false
	for _, p := range s.files {
		neg := strings.HasPrefix(p, "!")
		p = strings.TrimPrefix(p, "!")
		name := rel
		if !strings.Contains(p, "/") {
			name = path.Base(rel)
		}
		if ok, _ := doublestar.Match(p, name); ok {
			matched = !neg
		}
	}
	return matched
}
