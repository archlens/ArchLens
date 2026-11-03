package caching

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/archlens/ArchLens/graph"
	"github.com/archlens/ArchLens/parsers"
	"github.com/archlens/ArchLens/utils"
	"go.uber.org/zap"
)

var Sugar *zap.SugaredLogger

func init() {
	Sugar = utils.NewPrettySugaredLogger()
}

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
func (c *Cache) Refresh(root string, files map[string]struct{}, runCommand string, reParseFlag bool) error {
	head, herr := gitHead(root)
	changes, cerr := gitChanges(root, c.Commit)

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
	Sugar.Infof("parsed %d of %d files (full=%v)", len(list), len(files), full)

	if c.Files == nil || full {
		c.Files = map[string]Entry{}
	}
	for _, r := range parsers.GetASTs(list, root, runCommand) {
		if r.Err != nil {
			// TODO: Should abort instead of continuing
			delete(c.Files, r.File) // not cached, so it's retried next run
			Sugar.Errorf("%s: %v", r.File, r.Err)
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
func (c *Cache) Graph(files []string) *graph.Graph {
	g := graph.NewGraph()
	for _, f := range files {
		e, ok := c.Files[f]
		if !ok { // TODO: fails silently. Should probably say something about file not being in cache
			continue
		}
		deps := make([]graph.FileDependency, len(e.Dependencies))
		for i, d := range e.Dependencies {
			deps[i] = graph.FileDependency{Package: d.Package, File: d.File}
		}
		g.AddFile(e.Package, f, deps...)
	}
	return g
}

func LoadCache(path string) *Cache {
	data, err := os.ReadFile(path)
	if err != nil {
		return &Cache{}
	}
	var c Cache
	if json.Unmarshal(data, &c) != nil {
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
	defer func(){
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
	for _, file  := range c.Known {
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
func (r *RefCache) Load(root string) *Cache {
	best, bestN := &Cache{}, -1
	files, _ := filepath.Glob(filepath.Join(r.Dir, "*.json"))
	for _, f := range files {
		cand := LoadCache(f)
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

// cacheSchema changes when the shape of Entry changes, so cache files written
// by an older shape are never loaded.
const cacheSchema = "2"

func Fingerprint(version, runCommand string) string {
	sum := sha256.Sum256([]byte(cacheSchema + "\x00" + version + "\x00" + runCommand))
	return hex.EncodeToString(sum[:8])
}