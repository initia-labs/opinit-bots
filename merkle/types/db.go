package types

import "encoding/json"

type TreeInfo struct {
	// Index of the tree used as prefix for the keys
	Index uint64 `json:"index"`

	// Number of leaves in the tree
	LeafCount uint64 `json:"leaf_count"`

	// Cumulative number of leaves all the way up to the current tree
	StartLeafIndex uint64 `json:"start_leaf_index"`

	// Last sibling of the height(level) of the tree
	LastSiblings map[uint8][]byte `json:"height_data"`

	// Flag to indicate if the tree is finalized
	Done bool `json:"done"`
}

func (t TreeInfo) Key() []byte {
	return PrefixedWorkingTreeKey(t.Index)
}

func (t TreeInfo) Value() ([]byte, error) {
	return json.Marshal(t)
}

type FinalizedTreeInfo struct {
	// TreeIndex is the index of the tree used as prefix for the keys,
	// which is incremented by 1 for each new tree.
	TreeIndex  uint64 `json:"tree_index"`
	TreeHeight uint8  `json:"tree_height"`
	Root       []byte `json:"root"`
	// StartLeafIndex is the cumulative number of leaves all the way up to the current tree.
	// This approach helps to map the l2 withdrawal sequence to the tree index.
	StartLeafIndex uint64 `json:"start_leaf_index"`
	LeafCount      uint64 `json:"leaf_count"`
	ExtraData      []byte `json:"extra_data,omitempty"`
}

func (f FinalizedTreeInfo) Key() []byte {
	return PrefixedFinalizedTreeKey(f.StartLeafIndex)
}

func (f FinalizedTreeInfo) Value() ([]byte, error) {
	return json.Marshal(f)
}

type Node struct {
	// TreeIndex is the index of the tree used as prefix for the keys.
	TreeIndex uint64 `json:"tree_index"`
	// Height of the node in the tree
	Height uint8 `json:"height"`
	// LocalNodeIndex is the index of the node at the given height
	LocalNodeIndex uint64 `json:"local_node_index"`
	Data           []byte `json:"data"`
}

func (n Node) Key() []byte {
	return PrefixedNodeKey(n.TreeIndex, n.Height, n.LocalNodeIndex)
}

func (n Node) Value() []byte {
	return n.Data
}
