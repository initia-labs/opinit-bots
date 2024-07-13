package types

import (
	"encoding/binary"

	dbtypes "github.com/initia-labs/opinit-bots-go/db/types"
)

var (
	FinalizedTreeKey = []byte("finalized_tree")
	WorkingTreeKey   = []byte("working_tree")
	NodeKey          = []byte("node")
)

func GetNodeKey(treeIndex uint64, level uint8, levelIndex uint64) []byte {
	data := make([]byte, 17)
	binary.BigEndian.PutUint64(data, treeIndex)
	data[8] = level
	binary.BigEndian.PutUint64(data[9:], levelIndex)
	return data
}

func PrefixedNodeKey(treeIndex uint64, level uint8, levelIndex uint64) []byte {
	return append(append(NodeKey, dbtypes.Splitter), GetNodeKey(treeIndex, level, levelIndex)...)
}

func PrefixedFinalizedTreeKey(startLeafIndex uint64) []byte {
	return append(append(FinalizedTreeKey, dbtypes.Splitter), dbtypes.FromUint64Key(startLeafIndex)...)
}

func PrefixedWorkingTreeKey(version uint64) []byte {
	return append(append(WorkingTreeKey, dbtypes.Splitter), dbtypes.FromUint64Key(version)...)
}
