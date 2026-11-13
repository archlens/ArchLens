package html

import (
	"bytes"
	_ "embed"
	"html/template"

	"github.com/archlens/archlens/graph"
)

//go:embed template.html
var page string

var tmpl = template.Must(template.New("page").Parse(page))

type Inner interface {
	ExportGraph(g *graph.Graph, title, rootName string) []byte
	ExportDiff(d *graph.Diff, title, rootName string) []byte
}

type Exporter struct {
	Inner Inner
	Live  bool // adds the auto-reload snippet for watch mode
}

func (e Exporter) Extension() string { return ".html" }

func (e Exporter) ExportGraph(g *graph.Graph, title, rootName string) []byte {
	return e.wrap(title, e.Inner.ExportGraph(g, title, rootName))
}

func (e Exporter) ExportDiff(d *graph.Diff, title, rootName string) []byte {
	return e.wrap(title, e.Inner.ExportDiff(d, title, rootName))
}

func (e Exporter) wrap(title string, mermaid []byte) []byte {
	var buf bytes.Buffer
	_ = tmpl.Execute(&buf, struct {
		Title, Mermaid string
		Live           bool
	}{title, string(mermaid), e.Live})
	return buf.Bytes()
}
