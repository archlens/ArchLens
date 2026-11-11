package export

import (
	"os"
	"path/filepath"

	"github.com/archlens/archlens/graph"
)

type Exporter interface {
	Extension() string
	ExportGraph(g *graph.Graph, title, rootName string) []byte
	ExportDiff(d *graph.Diff, title, rootName	 string) []byte
}

func Save(e Exporter, dir, name string, data []byte) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, name+e.Extension())
	return path, os.WriteFile(path, data, 0o644)
}
