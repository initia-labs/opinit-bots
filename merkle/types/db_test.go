package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewTreeInfo(t *testing.T) {
	tree := NewTreeInfo(1, 2, 3, map[uint8][]byte{1: {0x1}, 2: {0x2}}, true)
	require.Equal(t, uint64(1), tree.Index)
	require.Equal(t, uint64(2), tree.LeafCount)
	require.Equal(t, uint64(3), tree.StartLeafIndex)
	require.Equal(t, map[uint8][]byte{1: {0x1}, 2: {0x2}}, tree.LastSiblings)
	require.True(t, tree.Done)
}

func TestTreeKey(t *testing.T) {
	tree := NewTreeInfo(1, 2, 3, map[uint8][]byte{1: {0x1}, 2: {0x2}}, true)
	require.Equal(t, append(WorkingTreePrefix, []byte{byte('/'), 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1}...), tree.Key())
}

func TestTreeValue(t *testing.T) {
	tree := NewTreeInfo(1, 2, 3, map[uint8][]byte{1: {0x1}, 2: {0x2}}, true)
	bz, err := tree.Value()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"index":1,"leaf_count":2,"start_leaf_index":3,"height_data":{"1":"AQ==","2":"Ag=="},"done":true}`), bz)
}

func TestTreeMarshal(t *testing.T) {
	tree := NewTreeInfo(1, 2, 3, map[uint8][]byte{1: {0x1}, 2: {0x2}}, true)
	bz, err := tree.Marshal()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"index":1,"leaf_count":2,"start_leaf_index":3,"height_data":{"1":"AQ==","2":"Ag=="},"done":true}`), bz)
}

func TestTreeUnmarshal(t *testing.T) {
	bz := []byte(`{"index":1,"leaf_count":2,"start_leaf_index":3,"height_data":{"1":"AQ==","2":"Ag=="},"done":true}`)
	tree := &TreeInfo{}
	err := tree.Unmarshal(bz)
	require.NoError(t, err)

	require.Equal(t, uint64(1), tree.Index)
	require.Equal(t, uint64(2), tree.LeafCount)
	require.Equal(t, uint64(3), tree.StartLeafIndex)
	require.Equal(t, map[uint8][]byte{1: {0x1}, 2: {0x2}}, tree.LastSiblings)
	require.True(t, tree.Done)

	bz = []byte("")
	tree = &TreeInfo{}
	err = tree.Unmarshal(bz)
	require.Error(t, err)
}

func TestNewFinalizedTreeInfo(t *testing.T) {
	tree := NewFinalizedTreeInfo(1, 2, []byte{0x1}, 3, 4, []byte{0x2})
	require.Equal(t, uint64(1), tree.TreeIndex)
	require.Equal(t, uint8(2), tree.TreeHeight)
	require.Equal(t, []byte{0x1}, tree.Root)
	require.Equal(t, uint64(3), tree.StartLeafIndex)
	require.Equal(t, uint64(4), tree.LeafCount)
	require.Equal(t, []byte{0x2}, tree.ExtraData)
}

func TestFinalizedTreeKey(t *testing.T) {
	tree := NewFinalizedTreeInfo(1, 2, []byte{0x1}, 3, 4, []byte{0x2})
	require.Equal(t, append(FinalizedTreePrefix, []byte{byte('/'), 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x3}...), tree.Key())
}

func TestFinalizedTreeValue(t *testing.T) {
	tree := NewFinalizedTreeInfo(1, 2, []byte{0x1}, 3, 4, []byte{0x2})
	bz, err := tree.Value()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"tree_index":1,"tree_height":2,"root":"AQ==","start_leaf_index":3,"leaf_count":4,"extra_data":"Ag=="}`), bz)
}

func TestFinalizedTreeMarshal(t *testing.T) {
	tree := NewFinalizedTreeInfo(1, 2, []byte{0x1}, 3, 4, []byte{0x2})
	bz, err := tree.Marshal()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"tree_index":1,"tree_height":2,"root":"AQ==","start_leaf_index":3,"leaf_count":4,"extra_data":"Ag=="}`), bz)
}

func TestFinalizedTreeUnmarshal(t *testing.T) {
	bz := []byte(`{"tree_index":1,"tree_height":2,"root":"AQ==","start_leaf_index":3,"leaf_count":4,"extra_data":"Ag=="}`)
	tree := &FinalizedTreeInfo{}
	err := tree.Unmarshal(bz)
	require.NoError(t, err)

	require.Equal(t, uint64(1), tree.TreeIndex)
	require.Equal(t, uint8(2), tree.TreeHeight)
	require.Equal(t, []byte{0x1}, tree.Root)
	require.Equal(t, uint64(3), tree.StartLeafIndex)
	require.Equal(t, uint64(4), tree.LeafCount)
	require.Equal(t, []byte{0x2}, tree.ExtraData)

	bz = []byte("")
	tree = &FinalizedTreeInfo{}
	err = tree.Unmarshal(bz)
	require.Error(t, err)
}

func TestNewNode(t *testing.T) {
	node := NewNode(1, 2, 3, []byte{0x1})
	require.Equal(t, uint64(1), node.TreeIndex)
	require.Equal(t, uint8(2), node.Height)
	require.Equal(t, uint64(3), node.LocalNodeIndex)
	require.Equal(t, []byte{0x1}, node.Data)
}

func TestNodeKey(t *testing.T) {
	node := NewNode(1, 2, 3, []byte{0x1})
	require.Equal(t, append(NodePrefix, []byte{byte('/'), 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x3}...), node.Key())
}

func TestNodeValue(t *testing.T) {
	node := NewNode(1, 2, 3, []byte{0x1})
	bz := node.Value()
	require.Equal(t, []byte{0x1}, bz)
}
