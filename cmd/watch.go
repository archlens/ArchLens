package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/archlens/archlens/export/html"
	"github.com/archlens/archlens/export/mermaid"
	"github.com/archlens/archlens/serve"
)

// runWatch serves one view on localhost and rebuilds it when project files change.
func runWatch(ctx context.Context, p *project, view string, port int, full bool) error {
	if view == "" {
		names := p.viewNames()
		if len(names) != 1 {
			return fmt.Errorf("multiple views configured, pick one with --view (%v)", names)
		}
		view = names[0]
	}
	if _, ok := p.cfg.Views[view]; !ok {
		return fmt.Errorf("unknown view %q (available: %v)", view, p.viewNames())
	}

	live := html.Exporter{Inner: mermaid.NewExporter(false), Live: true}

	build := func(full bool) ([]byte, error) {
		viewFiles, err := p.refresh(full)
		if err != nil {
			return nil, err
		}
		files := viewFiles[view]
		if len(files) == 0 {
			return nil, fmt.Errorf("view %q matched no files", view)
		}
		g := p.graph(files)
		Sugar.Infof("%s: %d nodes, %d edges", view, g.TotalNodes, g.TotalEdges)
		return live.ExportGraph(g, view, p.cfg.Name), nil
	}

	page, err := build(full) // honor --parse-full only for the first build
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	ignore := map[string]bool{
		".git":                         true,
		".archlens":                    true,
		"node_modules":                 true,
		filepath.Base(p.cfg.SaveLocation): true, // "diagrams"
	}

	return serve.Run(ctx, fmt.Sprintf("127.0.0.1:%d", port), p.absRoot, ignore,
		func() ([]byte, error) { return build(false) }, page)
}