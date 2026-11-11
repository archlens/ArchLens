package mermaid

import (
	"cmp"
	"fmt"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/TyphonHill/go-mermaid/diagrams/flowchart"
	"github.com/archlens/archlens/graph"
)

type Exporter struct{ markdown bool }

const (
	colorIncrease = "#2da44e" // green
	colorDecrease = "#cf222e" // red

	fillAdded   = "#dafbe1" // light green node background
	fillRemoved = "#ffebe9" // light red node background
	textOnFill  = "#1f2328" // dark text, readable on both fills
)

func NewExporter(markdown bool) Exporter {
	return Exporter{markdown: markdown}
}

func (e Exporter) Extension() string {
	if e.markdown {
		return ".md"
	}
	return ".mmd"
}

func (e Exporter) wrap(diagram string) []byte {
	if e.markdown {
		return []byte("```mermaid\n" + diagram + "```")
	}
	return []byte(diagram)
}

// labels maps each package to its display label: the last path segment, or the
// full path when two packages would otherwise look the same.
func (m Exporter) labels(names []string, rootName string) map[string]string {
	count := make(map[string]int, len(names))
	for _, n := range names {
		count[path.Base(n)]++
	}
	out := make(map[string]string, len(names))
	for _, n := range names {
		switch n {
		case ".":
			out[n] = rootName
		default:
			out[n] = n
		}
	}
	return out
}

// Generate standard mermaid diagram from graph
func (m Exporter) ExportGraph(g *graph.Graph, title, rootName string) []byte {
	fc := flowchart.NewFlowchart()
	fc.Title = title
	fc.SetDirection(flowchart.FlowchartDirectionLeftRight)

	names := g.GetNodeNames()
	slices.Sort(names)
	lbl := m.labels(names, rootName)

	nodes := make(map[string]*flowchart.Node, len(names))
	for _, name := range names {
		nodes[name] = fc.AddNode(lbl[name])
	}

	// Sort edges to ensure same diagram gets generated when using same data.
	edges := g.Edges()
	slices.SortFunc(edges, func(a, b graph.Edge) int {
		return cmp.Or(cmp.Compare(a.From, b.From), cmp.Compare(a.To, b.To))
	})
	for _, e := range edges {
		link := fc.AddLink(nodes[e.From], nodes[e.To])
		link.SetText(strconv.FormatUint(uint64(e.Count), 10))
	}
	return m.wrap(fc.String())
}

// nodeStyle returns the style for an added or removed package, or nil if unchanged.
func nodeStyle(d *graph.Diff, name string) *flowchart.NodeStyle {
	var fill, stroke string
	switch {
	case d.Added[name]:
		fill, stroke = fillAdded, colorIncrease
	case d.Removed[name]:
		fill, stroke = fillRemoved, colorDecrease
	default:
		return nil
	}
	s := flowchart.NewNodeStyle()
	s.Fill, s.Stroke, s.Color, s.StrokeWidth = fill, stroke, textOnFill, 2
	return s
}

// diffLabel is the link text: the current count, plus the change when there is one.
func diffLabel(e graph.EdgeDiff) string {
	switch d := e.Delta(); {
	case d > 0:
		return fmt.Sprintf(`"%d (+%d)"`, e.After, d)
	case d < 0:
		return fmt.Sprintf(`"%d (-%d)"`, e.After, -d)
	default:
		return fmt.Sprintf(`"%d"`, e.After)
	}
}

// ExportDiff converts a graph diff into a flowchart. Packages that were added are
// green and packages that were removed are red. Links whose dependency count
// went up are green, links where it went down are red, unchanged links keep the
// default style.
func (m Exporter) ExportDiff(d *graph.Diff, title, rootName string) []byte {
	fc := flowchart.NewFlowchart()
	fc.Title = title
	fc.SetDirection(flowchart.FlowchartDirectionLeftRight)

	lbl := m.labels(d.Nodes, rootName)
	nodes := make(map[string]*flowchart.Node, len(d.Nodes))
	graphNodes := d.Nodes
	slices.Sort(graphNodes)
	for _, name := range graphNodes {
		nodes[name] = fc.AddNode(lbl[name])
		if st := nodeStyle(d, name); st != nil {
			nodes[name].SetStyle(st)
		}
	}

	var styles strings.Builder
	edges := d.Edges
	slices.SortFunc(edges, func(a, b graph.EdgeDiff) int {
		return cmp.Or(cmp.Compare(a.From, b.From), cmp.Compare(a.To, b.To))
	})
	for i, e := range d.Edges {
		fc.AddLink(nodes[e.From], nodes[e.To]).SetText(diffLabel(e))

		color := ""
		if delta := e.Delta(); delta > 0 {
			color = colorIncrease
		} else if delta < 0 {
			color = colorDecrease
		}
		if color != "" {
			fmt.Fprintf(&styles, "    linkStyle %d stroke:%s,stroke-width:2px,color:%s\n", i, color, color)
		}
	}
	return m.wrap(fc.String() + styles.String())
}
