package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrefixedFinalizedTreeKey(t *testing.T) {
	startLeafIndex := uint64(256)
	key := PrefixedFinalizedTreeKey(startLeafIndex)
	require.Equal(t, key, append(FinalizedTreePrefix, []byte{byte('/'), 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x0}...))
}

func TestPrefixedWorkingTreeKey(t *testing.T) {
	treeIndex := uint64(256)
	key := PrefixedWorkingTreeKey(treeIndex)
	require.Equal(t, key, append(WorkingTreePrefix, []byte{byte('/'), 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x0}...))
}

func TestGetNodeKey(t *testing.T) {
	treeIndex := uint64(256)
	height := uint8(3)
	nodeIndex := uint64(16)

	key := GetNodeKey(treeIndex, height, nodeIndex)
	require.Equal(t, key, []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x0, 0x3, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x10})
}

func TestPrefixedNodeKey(t *testing.T) {
	treeIndex := uint64(256)
	height := uint8(3)
	nodeIndex := uint64(16)

	key := PrefixedNodeKey(treeIndex, height, nodeIndex)
	require.Equal(t, key, append(NodePrefix, []byte{byte('/'), 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x0, 0x3, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x10}...))
}

func TestParseFinalizedTreeKey(t *testing.T) {
	startLeafIndex := uint64(256)
	key := PrefixedFinalizedTreeKey(startLeafIndex)
	parsedStartLeafIndex, err := ParseFinalizedTreeKey(key)
	require.NoError(t, err)
	require.Equal(t, startLeafIndex, parsedStartLeafIndex)
}

func TestParseWorkingTreeKey(t *testing.T) {
	treeIndex := uint64(256)
	key := PrefixedWorkingTreeKey(treeIndex)
	parsedTreeIndex, err := ParseWorkingTreeKey(key)
	require.NoError(t, err)
	require.Equal(t, treeIndex, parsedTreeIndex)
}

func TestParseNodeKey(t *testing.T) {
	treeIndex := uint64(256)
	height := uint8(3)
	nodeIndex := uint64(16)

	key := PrefixedNodeKey(treeIndex, height, nodeIndex)
	parsedTreeIndex, parsedHeight, parsedNodeIndex, err := ParseNodeKey(key)
	require.NoError(t, err)
	require.Equal(t, treeIndex, parsedTreeIndex)
	require.Equal(t, height, parsedHeight)
	require.Equal(t, nodeIndex, parsedNodeIndex)
}
