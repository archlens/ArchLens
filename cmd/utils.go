package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/archlens/archlens/caching"
	"github.com/archlens/archlens/export"
	"github.com/archlens/archlens/export/html"
	"github.com/archlens/archlens/export/json"
	"github.com/archlens/archlens/export/mermaid"
	"github.com/archlens/archlens/graph"
	"github.com/archlens/archlens/input"
)

type project struct {
	cfg       *input.Input
	cfgDir    string
	absRoot   string
	selectors map[string]*input.Selector // one per view
	cache     *caching.Cache
}

const formatHelp = "Set the output format for archlens [mermaid|mermaid-md|json|html]"

// newExporter maps a --format value to an exporter. Shared by render and render-diff.
func newExporter(format string) (export.Exporter, error) {
	switch format {
	case "mermaid":
		return mermaid.NewExporter(false), nil
	case "mermaid-md":
		return mermaid.NewExporter(true), nil
	case "json":
		return json.Exporter{}, nil
	case "html":
		return html.Exporter{Inner: mermaid.NewExporter(false)}, nil
	default:
		return nil, fmt.Errorf("'%s' is not a valid export format", format)
	}
}

// refresh updates the working-tree cache on disk. Used by render and watch.
func (p *project) refresh(full bool) (map[string][]string, error) {
	if p.cache == nil {
		p.cache = caching.LoadCache(p.cachePath(), p.cfg.RunCommand)
	}
	viewFiles, err := p.parse(p.cache, p.absRoot, full)
	if err != nil {
		return nil, err
	}
	if err := p.cache.Save(p.cachePath()); err != nil {
		Sugar.Errorf("Error saving cache: %v", err)
	}
	return viewFiles, nil
}

// loadProject loads the config and resolves its directories. It touches no cache.
func loadProject(configPath string) (*project, error) {
	cfg, err := input.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("loading configuration: %w", err)
	}
	Sugar.Infof("Config: %+v", *cfg)

	absCfg, err := filepath.Abs(configPath)
	if err != nil {
		return nil, fmt.Errorf("resolving config location: %w", err)
	}
	cfgDir := filepath.Dir(absCfg)

	sels, err := cfg.Selectors()
	if err != nil {
		return nil, err
	}

	return &project{
		cfg:     cfg,
		cfgDir:  cfgDir,
		selectors: sels,
		absRoot: resolveRoot(cfgDir, cfg.RootFolder),
	}, nil
}

// graph builds the dependency graph for one view's files from cached parse
// results. Call refresh first, since that is what loads the cache.
// graph builds a view's graph from cached parse results.
func (p *project) graph(view string, files []string) *graph.Graph {
	return p.project(view, p.cache.Graph(files, p.absRoot))
}

// project shapes a full graph into what a view shows: sub-packages merged into
// the include entries, everything else dropped.
func (p *project) project(view string, g *graph.Graph) *graph.Graph {
	return g.Project(p.selectors[view].Node)
}

// parse resolves every view's files under root and brings c up to date for them.
// It works on any tree (the working tree or a git worktree) and any cache.
func (p *project) parse(c *caching.Cache, root string, full bool) (map[string][]string, error) {
	viewFiles, union, err := input.Discover(root, p.selectors)
	if err != nil {
		return nil, fmt.Errorf("finding files: %w", err)
	}
	if err := c.Refresh(root, p.cfgDir, union, p.cfg.RunCommand, full); err != nil {
		return nil, fmt.Errorf("parsing: %w", err)
	}
	return viewFiles, nil
}

// viewNames returns the configured view names, sorted.
func (p *project) viewNames() []string {
	names := make([]string, 0, len(p.cfg.Views))
	for n := range p.cfg.Views {
		names = append(names, n)
	}

	return names
}

// resolveRoot returns the absolute project root. A relative rootFolder is
// relative to the config file's directory, not the shell's working directory.
func resolveRoot(_, rootFolder string) string {
	abs, err := filepath.Abs(rootFolder) // relative to cwd, as before
	if err != nil {
		return filepath.Clean(rootFolder)
	}
	return abs
}

func (p *project) cachePath() string {
	return filepath.Join(p.absRoot, ".archlens", "cache.json")
}