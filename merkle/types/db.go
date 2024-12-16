// DB Structure
//
// * WorkingTree (TreeInfo)
//   - Key: `working_tree/${version}`
//   - Description: Stores the tree information at the given chain height ('version').
//
// * FinalizedTree
//   - Key: `finalized_tree/${start_leaf_index}`
//   - Description: Stores the finalized tree starting from the given `start_leaf_index`,
//     which corresponds to the first L2 sequence number of the first withdrawal.
//
// * Node
//   - Key: `node/${tree_index}${height}${local_node_index}`
//   - Description: Stores node information at a specific position in the tree.
//   - `tree_index`: Incremental index identifying the tree.
//   - `height`: Vertical position of the node in the tree.
//   - `local_node_index`: Index of the node within the tree, representing the order in which the node was added.
package types

import (
	"encoding/json"

	"github.com/pkg/errors"
)

type LegacyTreeInfo struct {
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

func (t LegacyTreeInfo) Migrate(version uint64) TreeInfo {
	return TreeInfo{
		Version:        version,
		Index:          t.Index,
		LeafCount:      t.LeafCount,
		StartLeafIndex: t.StartLeafIndex,
		LastSiblings:   t.LastSiblings,
		Done:           t.Done,
	}
}

type TreeInfo struct {
	// Version represents the chain height. We store the tree information for each height to
	// maintain a record of the state of the tree at that specific height.
	Version uint64 `json:"version"`

	// Index of the tree used as prefix for the keys
	Index uint64 `json:"index"`

	// Number of leaves in the tree
	LeafCount uint64 `json:"leaf_count"`

	// Cumulative number of leaves all the way up to the current tree
	StartLeafIndex uint64 `json:"start_leaf_index"`

	// Last sibling of the height(level) of the tree
	LastSiblings map[uint8][]byte `json:"last_siblings"`

	// Flag to indicate if the tree is finalized
	Done bool `json:"done"`
}

func NewTreeInfo(version uint64, index uint64, leafCount uint64, startLeafIndex uint64, lastSiblings map[uint8][]byte, done bool) TreeInfo {
	return TreeInfo{
		Version:        version,
		Index:          index,
		LeafCount:      leafCount,
		StartLeafIndex: startLeafIndex,
		LastSiblings:   lastSiblings,
		Done:           done,
	}
}

func (t TreeInfo) Key() []byte {
	return PrefixedWorkingTreeKey(t.Version)
}

func (t TreeInfo) Value() ([]byte, error) {
	return t.Marshal()
}

func (t TreeInfo) Marshal() ([]byte, error) {
	bz, err := json.Marshal(&t)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal tree info")
	}
	return bz, nil
}

func (t *TreeInfo) Unmarshal(data []byte) error {
	err := json.Unmarshal(data, t)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal tree info")
	}
	return nil
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

func NewFinalizedTreeInfo(treeIndex uint64, treeHeight uint8, root []byte, startLeafIndex uint64, leafCount uint64, extraData []byte) FinalizedTreeInfo {
	return FinalizedTreeInfo{
		TreeIndex:      treeIndex,
		TreeHeight:     treeHeight,
		Root:           root,
		StartLeafIndex: startLeafIndex,
		LeafCount:      leafCount,
		ExtraData:      extraData,
	}
}

func (f FinalizedTreeInfo) Key() []byte {
	// Store the finalized tree information with the start leaf index as its prefix.
	// This makes it easier to retrieve proofs using the L2 sequence number of the withdrawal request.
	// For more details, see the `GetProofs()` function.
	return PrefixedFinalizedTreeKey(f.StartLeafIndex)
}

func (f FinalizedTreeInfo) Value() ([]byte, error) {
	return f.Marshal()
}

func (f FinalizedTreeInfo) Marshal() ([]byte, error) {
	bz, err := json.Marshal(&f)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal finalized tree info")
	}
	return bz, nil
}

func (f *FinalizedTreeInfo) Unmarshal(data []byte) error {
	err := json.Unmarshal(data, f)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal finalized tree info")
	}
	return nil
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

func NewNode(treeIndex uint64, height uint8, localNodeIndex uint64, data []byte) Node {
	return Node{
		TreeIndex:      treeIndex,
		Height:         height,
		LocalNodeIndex: localNodeIndex,
		Data:           data,
	}
}

func (n Node) Key() []byte {
	return PrefixedNodeKey(n.TreeIndex, n.Height, n.LocalNodeIndex)
}

func (n Node) Value() []byte {
	return n.Data
}
