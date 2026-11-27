package caching

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/archlens/archlens/parsers"
	"github.com/archlens/archlens/utils"
	"github.com/archlens/archlens/utils/gitutils"
)

// cacheSchemaVersion changes when the shape of Entry changes, so cache files
// written by an older shape are never loaded.
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

// Fingerprint changes whenever parse results could differ.
func Fingerprint(runCommand string) string {
	sum := sha256.Sum256([]byte(cacheSchemaVersion + "\x00" + runCommand))
	return hex.EncodeToString(sum[:8])
}

// LoadCache reads the cache at path. It returns an empty Cache if the file is
// missing, corrupt or was written with a different fingerprint.
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

// SameFiles reports whether files is exactly the set the cache was last
// refreshed with.
func (c *Cache) SameFiles(files map[string]struct{}) bool {
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

// ParserID changes whenever the parser script changes, invalidating the cache.
func ParserID(script string) (string, error) {
	data, err := os.ReadFile(script)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
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
		changes, err := gitutils.Changes(root, cand.Commit)
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
