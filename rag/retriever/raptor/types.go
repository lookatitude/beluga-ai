package raptor

// TreeNode represents a single node in the RAPTOR tree hierarchy.
// Leaf nodes correspond to original document chunks; internal nodes hold
// LLM-generated summaries of their children.
type TreeNode struct {
	// ID uniquely identifies this node within the tree.
	ID string
	// Level is the depth in the tree (0 = leaf / original document).
	Level int
	// Content is the text content of this node.
	Content string
	// Embedding is the vector embedding of Content.
	Embedding []float32
	// Children holds the IDs of child nodes that were clustered to
	// produce this summary node. Empty for leaf nodes.
	Children []string
	// ParentID is the ID of the parent summary node. Empty for root nodes.
	ParentID string
	// Metadata holds arbitrary key-value pairs associated with this node.
	Metadata map[string]any
}

// Tree holds the complete RAPTOR tree structure with nodes indexed by ID.
type Tree struct {
	// Nodes maps node IDs to their TreeNode values.
	Nodes map[string]*TreeNode
	// MaxLevel is the highest level in the tree.
	MaxLevel int
}

// AllNodes returns all nodes in the tree as a flat slice, suitable for
// collapsed tree search.
func (t *Tree) AllNodes() []*TreeNode {
	nodes := make([]*TreeNode, 0, len(t.Nodes))
	for _, n := range t.Nodes {
		nodes = append(nodes, n)
	}
	return nodes
}

// NodesAtLevel returns all nodes at the specified level.
func (t *Tree) NodesAtLevel(level int) []*TreeNode {
	var nodes []*TreeNode
	for _, n := range t.Nodes {
		if n.Level == level {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

// TreeConfig holds configuration for building a RAPTOR tree.
type TreeConfig struct {
	// MaxLevels is the maximum number of summary levels to build above
	// the leaf level. Default: 3.
	MaxLevels int
	// MinClusterSize is the minimum number of nodes required to form a
	// cluster. Clusters smaller than this are not summarized. Default: 2.
	MinClusterSize int
}

// DefaultTreeConfig returns a TreeConfig with sensible defaults.
func DefaultTreeConfig() TreeConfig {
	return TreeConfig{
		MaxLevels:      3,
		MinClusterSize: 2,
	}
}
