package graph

import (
	"cmp"
	"errors"
	"maps"
	"path"
	"slices"
)

type Graph struct {
	nodes      map[string]*node
	TotalNodes uint
	TotalEdges uint
}

type node struct {
	name       string
	path       string // directory of the package, relative to the root (empty if unknown)
	incoming   uint
	outgoing   uint
	successors map[string]*node // Name of node to node pointer
	weights    map[string]uint
	fileLinks  map[string]map[FileLink]struct{} // Name of successor to the file pairs behind the edge
}

// Package is a node of the graph: its name and the directory it lives in.
type Package struct {
	Name string
	Path string
}

// FileLink says that FromFile (in the source package) imports ToFile (in the
// target package). ToFile is the target package path when the parser can't
// tell which file is imported.
type FileLink struct {
	FromFile string
	ToFile   string
}

// FileDependency is one import found in a file: the package it points to and
// the file within that package.
type FileDependency struct {
	Package string
	File    string
}

// Edge is a directed edge: From depends on To.
type Edge struct {
	From  string
	To    string
	Count uint       // number of files in From that import To
	Files []FileLink // the file pairs behind the edge, sorted and unique
}

func NewGraph() *Graph {
	return &Graph{
		nodes:      map[string]*node{},
		TotalNodes: 0,
		TotalEdges: 0,
	}
}

func newNode(name string) *node {
	return &node{
		name:       name,
		incoming:   0,
		outgoing:   0,
		successors: map[string]*node{},
		weights:    map[string]uint{},
		fileLinks:  map[string]map[FileLink]struct{}{},
	}
}

func (g *Graph) GetNodeNames() []string {
	return slices.Collect(maps.Keys(g.nodes))
}

// Add nodes to graph. Nodes that already are in the graph are ignored
func (g *Graph) AddNodes(nodes ...string) {
	for _, nodeName := range nodes {
		if _, ok := g.nodes[nodeName]; !ok {
			g.nodes[nodeName] = newNode(nodeName)
			g.TotalNodes++
		}
	}
}

// Removes a given node from the graph recursively walking all children to make sure it is no longer in the graph
// Function doesn't return error as removing non existing nodes changes nothing so this function shouldn't be able to fail
func (g *Graph) RemoveNodes(nodes ...string) {
	for _, nodeName := range nodes {
		n, ok := g.nodes[nodeName]
		if !ok {
			continue
		}

		// Outgoing edges (n -> child): children lose an incoming edge.
		// A self-loop is counted here, so the parent walk below skips n.
		for _, child := range n.successors {
			child.incoming--
			g.TotalEdges--
		}

		// Incoming edges (parent -> n): remove n from every parent.
		// This scans the whole graph, which is fine for a rare operation.
		for _, parent := range g.nodes {
			if parent == n {
				continue
			}
			if _, ok := parent.successors[nodeName]; ok {
				delete(parent.successors, nodeName)
				delete(parent.weights, nodeName)
				delete(parent.fileLinks, nodeName)
				parent.outgoing--
				g.TotalEdges--
			}
		}

		delete(g.nodes, nodeName)
		g.TotalNodes--
	}
}

// PackagePath returns the directory of a package, relative to the root. The
// bool is false if the package isn't in the graph. The path is empty for
// packages added with AddPackage, which carries no file information.
func (g *Graph) PackagePath(name string) (string, bool) {
	n, ok := g.nodes[name]
	if !ok {
		return "", false
	}
	return n.path, true
}

// Packages returns every package with its name and path, sorted by name.
func (g *Graph) Packages() []Package {
	pkgs := make([]Package, 0, len(g.nodes))
	for name, n := range g.nodes {
		pkgs = append(pkgs, Package{Name: name, Path: n.path})
	}
	slices.SortFunc(pkgs, func(a, b Package) int { return cmp.Compare(a.Name, b.Name) })
	return pkgs
}

// Edges returns every edge in the graph.
func (g *Graph) Edges() []Edge {
	edges := make([]Edge, 0, g.TotalEdges)
	for name, n := range g.nodes {
		for succ := range n.successors {
			links := slices.Collect(maps.Keys(n.fileLinks[succ]))
			slices.SortFunc(links, func(a, b FileLink) int {
				return cmp.Or(cmp.Compare(a.FromFile, b.FromFile), cmp.Compare(a.ToFile, b.ToFile))
			})
			edges = append(edges, Edge{From: name, To: succ, Count: n.weights[succ], Files: links})
		}
	}
	return edges
}

// AddFile records that file (in package pkg) imports deps. Call it once per
// parsed file. It does what AddPackage does for the packages, and also keeps
// the (file, imported file) pairs on each edge and the path of every package
// it sees: pkg's path is the directory of file, and a dependency's path is the
// directory of its File (or File itself when File is the package path, which
// is what a parser reports when it can't tell the file). Edge weights still count files
// per package pair, so a file importing two files of the same package adds 1
// to the weight and 2 file links.
func (g *Graph) AddFile(pkg, file string, deps ...FileDependency) {
	pkgs := make([]string, 0, len(deps))
	for _, d := range deps {
		pkgs = append(pkgs, d.Package)
	}
	g.AddPackage(pkg, pkgs...)

	g.setPath(pkg, path.Dir(file)+"/")
	n := g.nodes[pkg]
	for _, d := range deps {
		if d.Package == pkg {
			continue // No loops
		}
		if d.File == d.Package {
			g.setPath(d.Package, d.Package+"/")
		} else {
			g.setPath(d.Package, path.Dir(d.File)+"/")
		}
		set := n.fileLinks[d.Package]
		if set == nil {
			set = map[FileLink]struct{}{}
			n.fileLinks[d.Package] = set
		}
		set[FileLink{FromFile: file, ToFile: d.File}] = struct{}{}
	}
}

// setPath sets a package's path unless it already has one.
func (g *Graph) setPath(name, p string) {
	if n, ok := g.nodes[name]; ok && n.path == "" {
		n.path = p
	}
}

// AddPackage records that one file in pkg imports deps. Call it once per parsed
// file: nodes are created as needed and each pkg -> dep edge weight goes up by 1.
// It keeps no file information; use AddFile for that.
func (g *Graph) AddPackage(pkg string, deps ...string) {
	g.AddNodes(pkg)

	seen := make(map[string]struct{}, len(deps))
	filtered := make([]string, 0, len(deps))
	for _, d := range deps {
		if d == pkg {
			// No loops
			continue
		}
		if _, dup := seen[d]; dup {
			continue
		}
		seen[d] = struct{}{}
		filtered = append(filtered, d)
	}
	_ = g.AddSuccessors(pkg, filtered...)
}

// Adds nodes as predecessors to the source node. Predecessor nodes that
// aren't already in the graph will be added.
//
// Returns an error if the source node doesn't exist in the graph
func (g *Graph) AddPredecessors(source string, predecessors ...string) error {
	var sourceNode *node
	var ok bool
	if sourceNode, ok = g.nodes[source]; !ok {
		return errors.New("source node is not in graph")
	}

	var preNode *node
	for _, predecessorNode := range predecessors {
		if preNode, ok = g.nodes[predecessorNode]; !ok {
			// Create and add node if not in graph
			preNode = newNode(predecessorNode)
			g.nodes[predecessorNode] = preNode
			g.TotalNodes++
		}

		if _, ok = preNode.successors[source]; !ok {
			preNode.successors[source] = sourceNode
			preNode.outgoing++
			sourceNode.incoming++
			g.TotalEdges++
		}
		preNode.weights[source]++
	}
	return nil
}

// Adds nodes as successors to the source node. Successor nodes that
// aren't already in the graph will be added.
//
// Returns an error if the source node doesn't exist in the graph
func (g *Graph) AddSuccessors(source string, successors ...string) error {
	var sourceNode *node
	var ok bool
	if sourceNode, ok = g.nodes[source]; !ok {
		return errors.New("source node is not in graph")
	}
	var sucNode *node
	for _, successorNode := range successors {
		if sucNode, ok = g.nodes[successorNode]; !ok {
			// Create and add node if not in graph
			sucNode = newNode(successorNode)
			g.nodes[successorNode] = sucNode
			g.TotalNodes++
		}

		if _, ok = sourceNode.successors[successorNode]; !ok {
			sourceNode.successors[successorNode] = sucNode
			sucNode.incoming++
			sourceNode.outgoing++
			g.TotalEdges++
		}
		sourceNode.weights[successorNode]++
	}
	return nil
}
