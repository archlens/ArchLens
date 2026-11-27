package pipeline

import (
	"maps"
	"os"
	"slices"

	"github.com/archlens/archlens/caching"
	"github.com/archlens/archlens/graph"
	"github.com/archlens/archlens/parsers"
	"github.com/archlens/archlens/utils"
	"github.com/archlens/archlens/utils/gitutils"
)

// Refresh brings c up to date for files, the set of files to parse (relative
// to root). Only files that changed since the cache was written are parsed,
// unless a full parse is needed or requested with reParse.
func Refresh(c *caching.Cache, root, cfgAbs string, files map[string]struct{}, runCommand string, reParse bool) error {
	head, herr := gitutils.Head(root)
	changes, cerr := gitutils.Changes(root, c.Commit)
	c.Fingerprint = caching.Fingerprint(runCommand)

	// Reparse everything if git can't tell us what changed, or if the set of
	// files changed (an added or removed file can change how imports resolve).
	full := herr != nil || cerr != nil || c.Commit == "" || !c.SameFiles(files) || reParse

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
		c.Files = map[string]caching.Entry{}
	}
	for _, r := range parsers.GetASTs(list, root, cfgAbs, runCommand) {
		if r.Err != nil {
			// TODO: Should abort instead of continuing
			delete(c.Files, r.File) // not cached, so it's retried next run
			utils.Sugar.Errorf("%s: %v", r.File, r.Err)
			continue
		}
		c.Files[r.File] = caching.Entry{Package: r.AST.Package, Dependencies: r.AST.Dependencies}
	}

	for f := range c.Files {
		if _, ok := files[f]; !ok {
			delete(c.Files, f)
		}
	}

	c.Commit, c.Known, c.Dirty = head, slices.Sorted(maps.Keys(files)), nil
	if dirty, err := gitutils.Changes(root, head); err == nil {
		for _, p := range dirty {
			if _, ok := files[p]; ok {
				c.Dirty = append(c.Dirty, p)
			}
		}
	}
	return nil
}

// Graph builds a view's graph from the cached results for files. keep decides
// which dependencies (by imported file) are included; nil keeps all of them.
// rootFolder is the tree the cache was parsed from. It's cheap, so do it per run.
func Graph(c *caching.Cache, files []string, rootFolder string, keep func(string) bool) *graph.Graph {
	restore := utils.WithWorkingDirectory(rootFolder)
	defer restore()

	g := graph.NewGraph()
	for _, f := range files {
		e, ok := c.Files[f]
		if !ok { // TODO: fails silently. Should probably say something about file not being in cache
			continue
		}
		deps := make([]graph.FileDependency, 0, len(e.Dependencies))
		for _, d := range e.Dependencies {
			if keep != nil && !keep(d.File) {
				continue
			}
			if info, err := os.Stat(d.File); err != nil {
				utils.Sugar.Errorf("Failed to stat file %s: %v", d.File, err)
			} else if info.IsDir() {
				d.File += "/"
			}
			deps = append(deps, graph.FileDependency{Package: d.Package, File: d.File})
		}
		g.AddFile(e.Package, f, deps...)
	}
	return g
}