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
	cfg       *input.Input // whatever input.Load returns
	absRoot   string
	cachePath string
	cache     *caching.Cache // whatever caching.LoadCache returns
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

// resolveViewFiles expands every view's include/exclude patterns under root.
// It returns the normalized files per view and the union of all of them.
func resolveViewFiles(cfg *input.Input, root string) (map[string][]string, map[string]struct{}, error) {
	viewFiles := make(map[string][]string, len(cfg.Views))
	union := make(map[string]struct{})
	for name, view := range cfg.Views {
		files, err := input.GetFiles(&view, root)
		if err != nil {
			return nil, nil, fmt.Errorf("getting files for view %s: %w", name, err)
		}
		for i, f := range files {
			files[i] = filepath.ToSlash(filepath.Clean(f))
			union[files[i]] = struct{}{}
		}
		viewFiles[name] = files
	}
	return viewFiles, union, nil
}

func (p *project) refresh(full bool) (map[string][]string, error) {
	viewFiles, union, err := resolveViewFiles(p.cfg, p.cfg.RootFolder)
	if err != nil {
		return nil, err
	}
	if err := p.cache.Refresh(p.absRoot, union, p.cfg.RunCommand, full); err != nil {
		return nil, fmt.Errorf("refreshing cache: %w", err)
	}
	if err := p.cache.Save(p.cachePath); err != nil {
		Sugar.Errorf("Error saving cache: %v", err)
	}
	return viewFiles, nil
}

// loadProject loads the config and the on-disk parse cache next to it.
func loadProject(configPath string) (*project, error) {
	cfg, err := input.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("loading configuration: %w", err)
	}
	Sugar.Infof("Config: %+v", *cfg)

	absRoot, err := filepath.Abs(cfg.RootFolder)
	if err != nil {
		return nil, fmt.Errorf("resolving root folder: %w", err)
	}
	cachePath := filepath.Join(absRoot, ".archlens", "cache.json")
	return &project{
		cfg:       cfg,
		absRoot:   absRoot,
		cachePath: cachePath,
		cache:     caching.LoadCache(cachePath, cfg.RunCommand),
	}, nil
}

// graph builds the dependency graph for one view's files from cached parse results.
func (p *project) graph(files []string) *graph.Graph {
	return p.cache.Graph(files, p.absRoot)
}

// viewNames returns the configured view names, sorted.
func (p *project) viewNames() []string {
	names := make([]string, 0, len(p.cfg.Views))
	for n := range p.cfg.Views {
		names = append(names, n)
	}

	return names
}
