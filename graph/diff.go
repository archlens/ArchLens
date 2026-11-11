package graph

import (
	"maps"
	"slices"
)

// EdgeDiff is one dependency edge (From depends on To) in two versions of a graph.
// Before or After is 0 when the edge only exists in one of them.
type EdgeDiff struct {
	From     string
	To       string
	FromPath string
	ToPath   string
	Before   uint
	After    uint
}

// Delta is the change in the number of dependencies (positive = more).
func (e EdgeDiff) Delta() int { return int(e.After) - int(e.Before) }

// Diff is the union of two graphs with per-edge before/after counts.
type Diff struct {
	Nodes   []string        // sorted
	Edges   []EdgeDiff      // sorted by (From, To)
	Added   map[string]bool // nodes only in head
	Removed map[string]bool // nodes only in base
}

// Compare diffs head against base: Before comes from base, After from head.
func Compare(base, head *Graph) *Diff {
	nodes := map[string]struct{}{}
	inBase := map[string]bool{}
	inHead := map[string]bool{}
	for _, n := range base.GetNodeNames() {
		nodes[n] = struct{}{}
		inBase[n] = true
	}
	for _, n := range head.GetNodeNames() {
		nodes[n] = struct{}{}
		inHead[n] = true
	}
	added, removed := map[string]bool{}, map[string]bool{}
	for n := range nodes {
		if inHead[n] && !inBase[n] {
			added[n] = true
		} else if inBase[n] && !inHead[n] {
			removed[n] = true
		}
	}

	type key struct{ from, to string }
	byKey := map[key]*EdgeDiff{}
	for _, e := range base.Edges() {
		byKey[key{e.From, e.To}] = &EdgeDiff{
			From:     e.From,
			To:       e.To,
			FromPath: base.nodes[e.From].path,
			ToPath:   base.nodes[e.To].path,
			Before:   e.Count,
		}
	}
	for _, e := range head.Edges() {
		k := key{e.From, e.To}
		if d, ok := byKey[k]; ok {
			d.After = e.Count
		} else {
			byKey[k] = &EdgeDiff{
				From:     e.From,
				To:       e.To,
				FromPath: head.nodes[e.From].path,
				ToPath:   head.nodes[e.To].path,
				After:    e.Count,
			}
		}
	}

	edges := make([]EdgeDiff, 0, len(byKey))
	for _, d := range byKey {
		edges = append(edges, *d)
	}
	return &Diff{
		Nodes:   slices.Collect(maps.Keys(nodes)),
		Edges:   edges,
		Added:   added,
		Removed: removed,
	}
}
