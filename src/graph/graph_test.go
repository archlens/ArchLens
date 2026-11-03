package graph

import (
	"testing"
)

func Test_Graph(t *testing.T) {
	graph := NewGraph()
	if len(graph.nodes) != 0 {
		t.Errorf("Number of nodes in empty graph is not 0\nExpected:\t%d\nActual\t%d\n", 0, len(graph.nodes))
	}
	nodes := []string{"1", "2", "3", "4", "5", "6"}
	graph.AddNodes(nodes...)
	if len(graph.nodes) != len(nodes) {
		t.Errorf("Incorrect number of nodes in graph\nExpected:\t%d\nActual\t%d\n", len(nodes), len(graph.nodes))
	}
	if len(graph.nodes) != int(graph.TotalNodes) {
		t.Errorf("TotalNodes is not the same as length of graph.nodes\nTotalNodes:\t%d\nActual:\t%d\n", graph.TotalNodes, len(graph.nodes))
	}

	oneSuccessors := []string{"2", "3", "7"}
	err := graph.AddSuccessors("1", oneSuccessors...)
	if err != nil {
		t.Fatal("Failed to find source node in graph")
	}
	if len(graph.nodes) != len(nodes)+1 {
		t.Errorf("Incorrect number of nodes in graph. AddSuccessor might not have added the new node\nExpected:\t%d\nActual\t%d\n", len(nodes)+1, len(graph.nodes))
	}
	if int(graph.TotalEdges) != len(oneSuccessors) {
		t.Errorf("Incorrect number of edges\nExpected:\t%d\nActual:\t%d\n", len(oneSuccessors), graph.TotalEdges)
	}

	fivePredecessors := []string{"2", "3", "8"}
	err = graph.AddPredecessors("5", fivePredecessors...)
	if err != nil {
		t.Fatal("Failed to find source node in graph")
	}
	if len(graph.nodes) != len(nodes)+2 {
		t.Errorf("Incorrect number of nodes in graph. AddPredecessors might not have added the new node\nExpected:\t%d\nActual\t%d\n", len(nodes)+1, len(graph.nodes))
	}
	if int(graph.TotalEdges) != len(oneSuccessors)+len(fivePredecessors) {
		t.Errorf("Incorrect number of edges\nExpected:\t%d\nActual:\t%d\n", len(oneSuccessors)+len(fivePredecessors), graph.TotalEdges)
	}

	// Removing a node that isn't in the graph should change nothing
	graph.RemoveNodes("does-not-exist")
	if len(graph.nodes) != 8 || graph.TotalNodes != 8 || graph.TotalEdges != 6 {
		t.Errorf("Removing a missing node changed the graph\nNodes: %d, TotalNodes: %d, TotalEdges: %d\n",
			len(graph.nodes), graph.TotalNodes, graph.TotalEdges)
	}

	// Remove "5": it only has incoming edges (2, 3, 8 -> 5)
	graph.RemoveNodes("5")
	if _, ok := graph.nodes["5"]; ok {
		t.Error("Node 5 is still in graph.nodes after removal")
	}
	if len(graph.nodes) != 7 || graph.TotalNodes != 7 {
		t.Errorf("Incorrect node count after removing 5\nExpected:\t%d\nActual:\t%d (TotalNodes: %d)\n",
			7, len(graph.nodes), graph.TotalNodes)
	}
	if graph.TotalEdges != 3 {
		t.Errorf("Incorrect edge count after removing 5\nExpected:\t%d\nActual:\t%d\n", 3, graph.TotalEdges)
	}
	for _, node := range fivePredecessors {
		parent := graph.nodes[node]
		if _, ok := parent.successors["5"]; ok {
			t.Errorf("Node %s still has a dangling successor pointer to removed node 5", node)
		}
	}
}
