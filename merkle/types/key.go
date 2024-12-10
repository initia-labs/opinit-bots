package types

import (
	"encoding/binary"
	"fmt"

	dbtypes "github.com/initia-labs/opinit-bots/db/types"
)

var (
	FinalizedTreePrefix = []byte("finalized_tree")
	WorkingTreePrefix   = []byte("working_tree")
	NodePrefix          = []byte("node")

	FinalizedTreeKeyLength = len(FinalizedTreePrefix) + 1 + 8
	WorkingTreeKeyLength   = len(WorkingTreePrefix) + 1 + 8
	NodeKeyLength          = len(NodePrefix) + 1 + 8 + 1 + 8
)

func PrefixedFinalizedTreeKey(startLeafIndex uint64) []byte {
	return dbtypes.GenerateKey([][]byte{
		FinalizedTreePrefix,
		dbtypes.FromUint64Key(startLeafIndex),
	})
}

func PrefixedWorkingTreeKey(version uint64) []byte {
	return dbtypes.GenerateKey([][]byte{
		WorkingTreePrefix,
		dbtypes.FromUint64Key(version),
	})
}

func GetNodeKey(treeIndex uint64, height uint8, nodeIndex uint64) []byte {
	data := make([]byte, 17)
	binary.BigEndian.PutUint64(data, treeIndex)
	data[8] = height
	binary.BigEndian.PutUint64(data[9:], nodeIndex)
	return data
}

func PrefixedNodeKey(treeIndex uint64, height uint8, nodeIndex uint64) []byte {
	return dbtypes.GenerateKey([][]byte{
		NodePrefix,
		GetNodeKey(treeIndex, height, nodeIndex),
	})
}

func ParseFinalizedTreeKey(key []byte) (startLeafIndex uint64, err error) {
	if len(key) != FinalizedTreeKeyLength {
		return 0, fmt.Errorf("invalid finalized tree key bytes: expected %d, got %d", FinalizedTreeKeyLength, len(key))
	}
	cursor := len(FinalizedTreePrefix) + 1

	startLeafIndex = dbtypes.ToUint64Key(key[cursor : cursor+8])
	return
}

func ParseWorkingTreeKey(key []byte) (version uint64, err error) {
	if len(key) != WorkingTreeKeyLength {
		return 0, fmt.Errorf("invalid working tree key bytes: expected %d, got %d", WorkingTreeKeyLength, len(key))
	}
	cursor := len(WorkingTreePrefix) + 1

	version = dbtypes.ToUint64Key(key[cursor : cursor+8])
	return
}

func ParseNodeKey(key []byte) (treeIndex uint64, height uint8, nodeIndex uint64, err error) {
	if len(key) != NodeKeyLength {
		return 0, 0, 0, fmt.Errorf("invalid node key bytes: expected %d, got %d", NodeKeyLength, len(key))
	}
	cursor := len(NodePrefix) + 1

	treeIndex = binary.BigEndian.Uint64(key[cursor : cursor+8])
	cursor += 8

	height = key[cursor]
	cursor += 1

	nodeIndex = binary.BigEndian.Uint64(key[cursor : cursor+8])
	return
}
