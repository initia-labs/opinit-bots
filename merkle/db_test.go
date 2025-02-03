package merkle

import (
	"testing"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/initia-labs/opinit-bots/db"
	merkletypes "github.com/initia-labs/opinit-bots/merkle/types"
	"github.com/stretchr/testify/require"
)

func TestSaveGetWorkingTree(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)

	_, err = GetWorkingTree(db, 3)
	require.Error(t, err)

	workingTree := merkletypes.TreeInfo{
		Version:        10,
		Index:          3,
		LeafCount:      10,
		StartLeafIndex: 5,
		LastSiblings: map[uint8][]byte{
			0: []byte("node1"),
		},
		Done: true,
	}
	err = SaveWorkingTree(db, workingTree)
	require.NoError(t, err)

	tree, err := GetWorkingTree(db, 10)
	require.NoError(t, err)
	require.Equal(t, workingTree, tree)
}

func TestSaveGetFinalizedTree(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)

	_, err = GetFinalizedTree(db, 5)
	require.Error(t, err)

	finalizedTree := merkletypes.FinalizedTreeInfo{
		TreeIndex:      5,
		TreeHeight:     3,
		Root:           []byte("root"),
		StartLeafIndex: 5,
		LeafCount:      10,
		ExtraData:      []byte("extra data"),
	}
	err = SaveFinalizedTree(db, finalizedTree)
	require.NoError(t, err)

	tree, err := GetFinalizedTree(db, 5)
	require.NoError(t, err)
	require.Equal(t, finalizedTree, tree)
}

func TestSaveGetNodes(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)

	_, err = GetNodeBytes(db, 3, 5, 0)
	require.Error(t, err)
	_, err = GetNodeBytes(db, 3, 5, 1)
	require.Error(t, err)

	node0 := merkletypes.Node{
		TreeIndex:      3,
		Height:         5,
		LocalNodeIndex: 0,
		Data:           []byte("node0"),
	}
	node1 := merkletypes.Node{
		TreeIndex:      3,
		Height:         5,
		LocalNodeIndex: 1,
		Data:           []byte("node1"),
	}

	err = SaveNodes(db, node0, node1)
	require.NoError(t, err)

	node0bytes, err := GetNodeBytes(db, 3, 5, 0)
	require.NoError(t, err)

	require.Equal(t, node0.Value(), node0bytes)
	node1bytes, err := GetNodeBytes(db, 3, 5, 1)
	require.NoError(t, err)
	require.Equal(t, node1.Value(), node1bytes)
}

func TestGetProofs(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)

	hashFn := ophosttypes.GenerateNodeHash
	m, err := NewMerkle(hashFn)
	require.NoError(t, err)

	require.NoError(t, m.InitializeWorkingTree(10, 1, 1))

	// insert 6 nodes
	nodes, err := m.InsertLeaf([]byte("node1"))
	require.NoError(t, err)
	err = SaveNodes(db, nodes...)
	require.NoError(t, err)
	nodes, err = m.InsertLeaf([]byte("node2"))
	require.NoError(t, err)
	err = SaveNodes(db, nodes...)
	require.NoError(t, err)
	nodes, err = m.InsertLeaf([]byte("node3"))
	require.NoError(t, err)
	err = SaveNodes(db, nodes...)
	require.NoError(t, err)
	nodes, err = m.InsertLeaf([]byte("node4"))
	require.NoError(t, err)
	err = SaveNodes(db, nodes...)
	require.NoError(t, err)
	nodes, err = m.InsertLeaf([]byte("node5"))
	require.NoError(t, err)
	err = SaveNodes(db, nodes...)
	require.NoError(t, err)
	nodes, err = m.InsertLeaf([]byte("node6"))
	require.NoError(t, err)
	err = SaveNodes(db, nodes...)
	require.NoError(t, err)

	hash12 := hashFn([]byte("node1"), []byte("node2"))
	hash34 := hashFn([]byte("node3"), []byte("node4"))
	hash56 := hashFn([]byte("node5"), []byte("node6"))
	hash66 := hashFn([]byte("node6"), []byte("node6"))
	hash1234 := hashFn(hash12[:], hash34[:])
	hash5666 := hashFn(hash56[:], hash66[:])
	hashRoot := hashFn(hash1234[:], hash5666[:])

	extraData := []byte("extra data")
	finalizedTree, nodes, root, err := m.FinalizeWorkingTree(extraData)
	require.NoError(t, err)
	require.Equal(t, hashRoot[:], root)

	err = SaveFinalizedTree(db, *finalizedTree)
	require.NoError(t, err)
	err = SaveNodes(db, nodes...)
	require.NoError(t, err)

	proofs, treeIndex, root_, extraData, err := GetProofs(db, 1)
	require.NoError(t, err)
	require.Equal(t, uint64(1), treeIndex)
	require.Equal(t, root, root_)
	require.Equal(t, []byte("extra data"), extraData)
	require.Len(t, proofs, 3)
	require.Equal(t, []byte("node2"), proofs[0])
	require.Equal(t, hash34[:], proofs[1])
	require.Equal(t, hash5666[:], proofs[2])
}

func TestDeleteFutureFinalizedTrees(t *testing.T) { //nolint
	db, err := db.NewMemDB()
	require.NoError(t, err)

	for i := 1; i <= 10; i++ {
		finalizedTree := merkletypes.FinalizedTreeInfo{StartLeafIndex: uint64(i)}
		err = SaveFinalizedTree(db, finalizedTree)
		require.NoError(t, err)
	}

	err = DeleteFutureFinalizedTrees(db, 11)
	require.NoError(t, err)
	for i := 1; i <= 10; i++ {
		tree, err := GetFinalizedTree(db, uint64(i))
		require.NoError(t, err)
		require.Equal(t, tree.StartLeafIndex, uint64(i))
	}

	err = DeleteFutureFinalizedTrees(db, 5)
	require.NoError(t, err)
	for i := 1; i <= 4; i++ {
		tree, err := GetFinalizedTree(db, uint64(i))
		require.NoError(t, err)
		require.Equal(t, tree.StartLeafIndex, uint64(i))
	}
	for i := 5; i <= 10; i++ {
		_, err := GetFinalizedTree(db, uint64(i))
		require.Error(t, err)
	}

	err = DeleteFutureFinalizedTrees(db, 0)
	require.NoError(t, err)
	for i := 1; i <= 10; i++ {
		_, err := GetFinalizedTree(db, uint64(i))
		require.Error(t, err)
	}
}

func TestDeleteFutureWorkingTrees(t *testing.T) { //nolint
	db, err := db.NewMemDB()
	require.NoError(t, err)

	for i := 1; i <= 10; i++ {
		tree := merkletypes.TreeInfo{Version: uint64(i)}
		err = SaveWorkingTree(db, tree)
		require.NoError(t, err)
	}

	err = DeleteFutureWorkingTrees(db, 11)
	require.NoError(t, err)
	for i := 1; i <= 10; i++ {
		tree, err := GetWorkingTree(db, uint64(i))
		require.NoError(t, err)
		require.Equal(t, tree.Version, uint64(i))
	}

	err = DeleteFutureWorkingTrees(db, 5)
	require.NoError(t, err)
	for i := 1; i <= 4; i++ {
		tree, err := GetWorkingTree(db, uint64(i))
		require.NoError(t, err)
		require.Equal(t, tree.Version, uint64(i))
	}
	for i := 5; i <= 10; i++ {
		_, err := GetWorkingTree(db, uint64(i))
		require.Error(t, err)
	}

	err = DeleteFutureWorkingTrees(db, 0)
	require.NoError(t, err)
	for i := 1; i <= 10; i++ {
		_, err := GetWorkingTree(db, uint64(i))
		require.Error(t, err)
	}
}

func TestDeleteFutureNodes(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)

	var nodes []merkletypes.Node
	for treeIndex := uint64(1); treeIndex <= 10; treeIndex++ {
		for treeHeight := uint8(0); treeHeight <= 5; treeHeight++ {
			for nodeIndex := uint64(0); nodeIndex <= 10; nodeIndex++ {
				node := merkletypes.Node{
					TreeIndex:      treeIndex,
					Height:         treeHeight,
					LocalNodeIndex: nodeIndex,
					Data:           []byte{byte(treeIndex), treeHeight, byte(nodeIndex)},
				}
				nodes = append(nodes, node)
			}
		}
	}
	err = SaveNodes(db, nodes...)
	require.NoError(t, err)

	err = DeleteFutureNodes(db, 5)
	require.NoError(t, err)
	for treeIndex := uint64(1); treeIndex <= 4; treeIndex++ {
		for treeHeight := uint8(0); treeHeight <= 5; treeHeight++ {
			for nodeIndex := uint64(0); nodeIndex <= 10; nodeIndex++ {
				nodeBytes, err := GetNodeBytes(db, treeIndex, treeHeight, nodeIndex)
				require.NoError(t, err)
				require.Equal(t, []byte{byte(treeIndex), treeHeight, byte(nodeIndex)}, nodeBytes)
			}
		}
	}
	for treeIndex := uint64(5); treeIndex <= 10; treeIndex++ {
		for treeHeight := uint8(0); treeHeight <= 5; treeHeight++ {
			for nodeIndex := uint64(0); nodeIndex <= 10; nodeIndex++ {
				_, err := GetNodeBytes(db, treeIndex, treeHeight, nodeIndex)
				require.Error(t, err)
			}
		}
	}

	err = DeleteFutureNodes(db, 0)
	require.NoError(t, err)
	for treeIndex := uint64(0); treeIndex <= 10; treeIndex++ {
		for treeHeight := uint8(0); treeHeight <= 5; treeHeight++ {
			for nodeIndex := uint64(0); nodeIndex <= 10; nodeIndex++ {
				_, err := GetNodeBytes(db, treeIndex, treeHeight, nodeIndex)
				require.Error(t, err)
			}
		}
	}
}
