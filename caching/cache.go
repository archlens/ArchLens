package caching

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"maps"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/archlens/archlens/graph"
	"github.com/archlens/archlens/parsers"
	"github.com/archlens/archlens/utils"
	"github.com/bmatcuk/doublestar/v4"
)

// cacheSchema changes when the shape of Entry changes, so cache files written
// by an older shape are never loaded.
const cacheSchemaVersion = "3"

type Entry struct {
	Package      string               `json:"package"`
	Dependencies []parsers.Dependency `json:"dependencies"`
}

type Cache struct {
	Commit      string           `json:"commit"`
	Fingerprint string           `json:"fingerprint"`
	Known       []string         `json:"known"` // files parsed at the last refresh (sorted)
	Dirty       []string         `json:"dirty"` // files that differed from Commit at the last refresh
	Files       map[string]Entry `json:"files"`
}

// RefCache keeps one Cache per commit as <Dir>/<sha>.json.
type RefCache struct {
	Dir         string
	Keep        int    // newest files to keep; the rest are pruned on Save
	Fingerprint string // changes when parse results could differ (ArchLens version, runCommand)
}

type Change struct {
	Status byte   // 'M', 'A', 'D'
	Path   string // relative to root
}

// Filter selects packages by directory (relative to the project root).
// Patterns are globs and match recursively: "cmd" matches cmd and cmd/x/y.
type Filter struct {
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
}

// FilterSet accepts a package if any of its filters does.
type FilterSet []Filter

func matchDir(pattern, dir string) bool {
	pattern = path.Clean(filepath.ToSlash(pattern))
	if pattern == "." {
		return true // the root, recursively, is everything
	}
	if ok, _ := doublestar.Match(pattern, dir); ok {
		return true
	}
	ok, _ := doublestar.Match(pattern+"/**", dir)
	return ok
}

// Match reports whether the package directory dir is included and not excluded.
func (f Filter) Match(dir string) bool {
	dir = path.Clean(filepath.ToSlash(dir))
	if !slices.ContainsFunc(f.Include, func(p string) bool { return matchDir(p, dir) }) {
		return false
	}
	return !slices.ContainsFunc(f.Exclude, func(p string) bool { return matchDir(p, dir) })
}

func (s FilterSet) Match(dir string) bool {
	return slices.ContainsFunc(s, func(f Filter) bool { return f.Match(dir) })
}

// depDir is the directory of the package a dependency points to.
// Mirrors graph.AddFile: File == Package means File is the package path.
func depDir(d parsers.Dependency) string {
	if d.File == d.Package {
		return path.Clean(filepath.ToSlash(d.File))
	}
	return path.Dir(filepath.ToSlash(d.File))
}

// Keep returns the dependencies whose package is accepted by the set.
func (s FilterSet) Keep(deps []parsers.Dependency) []parsers.Dependency {
	kept := make([]parsers.Dependency, 0, len(deps))
	for _, d := range deps {
		if s.Match(depDir(d)) {
			kept = append(kept, d)
		}
	}
	return kept
}

// gitChanges lists paths (relative to root, forward slashes) that differ from
// commit, plus untracked files that git doesn't ignore.
func gitChanges(root, commit string) ([]string, error) {
	var paths []string
	for _, args := range [][]string{
		{"diff", "--name-only", "--no-renames", "--relative", "-z", commit, "--", "."},
		{"ls-files", "--others", "--exclude-standard", "-z"},
	} {
		out, err := exec.Command("git", append([]string{"-C", root}, args...)...).Output()
		if err != nil {
			return nil, err
		}
		for _, p := range bytes.Split(out, []byte{0}) {
			if len(p) > 0 {
				paths = append(paths, string(p))
			}
		}
	}
	return paths, nil
}

// gitHead returns the commit HEAD points to in the repo containing root.
func gitHead(root string) (string, error) {
	out, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// Refresh brings the cache up to date for files, which must be sorted and unique.
func (c *Cache) Refresh(root, cfgAbs string, files map[string]struct{}, runCommand string, filters FilterSet, reParseFlag bool) error {
	head, herr := gitHead(root)
	changes, cerr := gitChanges(root, c.Commit)
	c.Fingerprint = Fingerprint(runCommand, filters)

	// Reparse everything if git can't tell us what changed, or if the set of
	// files changed (an added or removed file can change how imports resolve).
	full := herr != nil || cerr != nil || c.Commit == "" || !c.sameFile(files) || reParseFlag

	toParse := map[string]bool{}
	if !full {
		for _, p := range changes {
			toParse[p] = true
		}
		for _, p := range c.Dirty {
			toParse[p] = true
		}
	}

	var list []string
	for f := range files {
		if _, cached := c.Files[f]; full || toParse[f] || !cached {
			list = append(list, f)
		}
	}
	utils.Sugar.Infof("parsed %d of %d files (full=%v)", len(list), len(files), full)

	if c.Files == nil || full {
		c.Files = map[string]Entry{}
	}
	for _, r := range parsers.GetASTs(list, root, cfgAbs, runCommand) {
		if r.Err != nil {
			// TODO: Should abort instead of continuing
			delete(c.Files, r.File) // not cached, so it's retried next run
			utils.Sugar.Errorf("%s: %v", r.File, r.Err)
			continue
		}
		c.Files[r.File] = Entry{Package: r.AST.Package, Dependencies: r.AST.Dependencies}
	}

	for f := range c.Files {
		if _, ok := files[f]; !ok {
			delete(c.Files, f)
		}
	}

	c.Commit, c.Known, c.Dirty = head, slices.Sorted(maps.Keys(files)), nil
	if dirty, err := gitChanges(root, head); err == nil {
		for _, p := range dirty {
			if _, ok := files[p]; ok {
				c.Dirty = append(c.Dirty, p)
			}
		}
	}
	return nil
}

// Graph builds a view's graph from cached results. It's cheap, so do it per run.
func (c *Cache) Graph(files []string, rootFolder string, filter FilterSet) *graph.Graph {
	restore := utils.WithWorkingDirectory(rootFolder)
	defer restore()

	g := graph.NewGraph()
	for _, f := range files {
		e, ok := c.Files[f]
		if !ok { // TODO: fails silently. Should probably say something about file not being in cache
			continue
		}
		kept := filter.Keep(e.Dependencies)
		deps := make([]graph.FileDependency, len(kept))
		for i, d := range kept {
			info, err := os.Stat(d.File)
			if err != nil {
				utils.Sugar.Errorf("Failed to stat file %s: %v", d.File, err)
			}
			if info.IsDir() {
				d.File += "/"
			}
			deps[i] = graph.FileDependency{Package: d.Package, File: d.File}
		}
		g.AddFile(e.Package, f, deps...)
	}
	return g
}

func LoadCache(path, fingerprint string) *Cache {
	data, err := os.ReadFile(path)
	if err != nil {
		return &Cache{}
	}
	var c Cache
	if json.Unmarshal(data, &c) != nil || c.Fingerprint != fingerprint {
		utils.Sugar.Debug("Failed to load cache or cache invalid. Returned empty cache")
		return &Cache{} // corrupt cache: start over
	}

	return &c
}

// Save writes atomically so a crash never leaves a half-written cache.
func (c *Cache) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "cache-*.tmp")
	if err != nil {
		return err
	}
	defer func() {
		_ = os.Remove(tmp.Name())
	}() // no-op after a successful rename
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// ParserID changes whenever the parser script changes, invalidating the cache.
func ParserID(script string) (string, error) {
	data, err := os.ReadFile(script)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func (c *Cache) sameFile(files map[string]struct{}) bool {
	if len(files) != len(c.Known) {
		return false
	}
	for _, file := range c.Known {
		if _, ok := files[file]; !ok {
			return false
		}
	}
	return true
}

func (r *RefCache) path(sha string) string {
	return filepath.Join(r.Dir, sha+".json")
}

// Load returns the cached parse closest to the checkout at root, or an empty Cache.
func (r *RefCache) Load(root, fingerprint string) *Cache {
	best, bestN := &Cache{}, -1
	files, _ := filepath.Glob(filepath.Join(r.Dir, "*.json"))
	for _, f := range files {
		cand := LoadCache(f, fingerprint)
		if cand.Commit == "" || cand.Fingerprint != r.Fingerprint {
			continue // empty, corrupt or from a different parser setup
		}
		changes, err := gitChanges(root, cand.Commit)
		if err != nil {
			continue // commit not in this repo
		}
		if bestN < 0 || len(changes) < bestN {
			best, bestN = cand, len(changes)
		}
	}
	return best
}

// Save stores c under its commit and prunes the oldest files beyond Keep.
func (r *RefCache) Save(c *Cache) error {
	if c.Commit == "" {
		return nil // git couldn't tell us the commit, so there's nothing to key on
	}
	c.Fingerprint = r.Fingerprint
	if err := c.Save(r.path(c.Commit)); err != nil {
		return err
	}
	files, _ := filepath.Glob(filepath.Join(r.Dir, "*.json"))
	sort.Slice(files, func(i, j int) bool {
		a, _ := os.Stat(files[i])
		b, _ := os.Stat(files[j])
		return a != nil && b != nil && a.ModTime().After(b.ModTime())
	})
	for _, f := range files[min(r.Keep, len(files)):] {
		_ = os.Remove(f)
	}
	return nil
}

func Fingerprint(runCommand string, filters FilterSet) string {
	var b strings.Builder
	field := func(s string) {
		b.WriteString(strconv.Itoa(len(s)))
		b.WriteByte(':')
		b.WriteString(s)
	}
	list := func(l []string) {
		b.WriteString(strconv.Itoa(len(l)))
		b.WriteByte('[')
		for _, s := range l {
			field(s)
		}
		b.WriteByte(']')
	}

	field(cacheSchemaVersion)
	field(runCommand)
	b.WriteString(strconv.Itoa(len(filters)))
	b.WriteByte(';')
	for _, f := range filters {
		list(f.Include)
		list(f.Exclude)
	}

	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:8])
}
