package merkle

import (
	"encoding/json"
	"testing"

	"golang.org/x/crypto/sha3"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/initia-labs/opinit-bots/db"
	merkletypes "github.com/initia-labs/opinit-bots/merkle/types"
	"github.com/stretchr/testify/require"
)

func Test_validateNodeGeneratorFn(t *testing.T) {
	fnNonCommutable := func(a, b []byte) [32]byte {
		return sha3.Sum256(append(a, b...))
	}

	fnCommutable := ophosttypes.GenerateNodeHash
	require.Error(t, validateNodeGeneratorFn(fnNonCommutable))
	require.NoError(t, validateNodeGeneratorFn(fnCommutable))
}

func Test_MerkleTree_LastSibling(t *testing.T) {
	tempDir := t.TempDir()
	db, err := db.NewDB(tempDir)
	require.NoError(t, err)

	hashFn := ophosttypes.GenerateNodeHash
	m, err := NewMerkle(db, hashFn)
	require.NoError(t, err)

	require.NoError(t, m.InitializeWorkingTree(1, 1))

	// empty tree
	require.Len(t, m.workingTree.LastSiblings, 0)

	// 1 node
	require.NoError(t, m.InsertLeaf([]byte("node1")))
	require.Len(t, m.workingTree.LastSiblings, 1)
	require.Equal(t, []byte("node1"), m.workingTree.LastSiblings[0])

	// 2 nodes
	hash12 := hashFn([]byte("node1"), []byte("node2"))
	require.NoError(t, m.InsertLeaf([]byte("node2")))
	require.Len(t, m.workingTree.LastSiblings, 2)
	require.Equal(t, []byte("node2"), m.workingTree.LastSiblings[0])
	require.Equal(t, hash12[:], m.workingTree.LastSiblings[1])

	// 3 nodes
	require.NoError(t, m.InsertLeaf([]byte("node3")))
	require.Len(t, m.workingTree.LastSiblings, 2)
	require.Equal(t, []byte("node3"), m.workingTree.LastSiblings[0])
	require.Equal(t, hash12[:], m.workingTree.LastSiblings[1])

	// 4 nodes
	hash34 := hashFn([]byte("node3"), []byte("node4"))
	hash1234 := hashFn(hash12[:], hash34[:])
	require.NoError(t, m.InsertLeaf([]byte("node4")))
	require.Len(t, m.workingTree.LastSiblings, 3)
	require.Equal(t, []byte("node4"), m.workingTree.LastSiblings[0])
	require.Equal(t, hash34[:], m.workingTree.LastSiblings[1])
	require.Equal(t, hash1234[:], m.workingTree.LastSiblings[2])

	// 5 nodes
	require.NoError(t, m.InsertLeaf([]byte("node5")))
	require.Len(t, m.workingTree.LastSiblings, 3)
	require.Equal(t, []byte("node5"), m.workingTree.LastSiblings[0])
	require.Equal(t, hash34[:], m.workingTree.LastSiblings[1])
	require.Equal(t, hash1234[:], m.workingTree.LastSiblings[2])

	// 6 nodes
	hash56 := hashFn([]byte("node5"), []byte("node6"))
	require.NoError(t, m.InsertLeaf([]byte("node6")))
	require.Len(t, m.workingTree.LastSiblings, 3)
	require.Equal(t, []byte("node6"), m.workingTree.LastSiblings[0])
	require.Equal(t, hash56[:], m.workingTree.LastSiblings[1])
	require.Equal(t, hash1234[:], m.workingTree.LastSiblings[2])
}

func Test_FinalizeWorkingTree(t *testing.T) {
	tempDir := t.TempDir()
	db, err := db.NewDB(tempDir)
	require.NoError(t, err)

	hashFn := ophosttypes.GenerateNodeHash
	m, err := NewMerkle(db, hashFn)
	require.NoError(t, err)

	require.NoError(t, m.InitializeWorkingTree(1, 1))

	// empty tree
	kvs, root, err := m.FinalizeWorkingTree(nil)
	require.NoError(t, err)
	require.Len(t, kvs, 0)
	require.Equal(t, merkletypes.EmptyRootHash[:], root)

	// insert 6 nodes
	require.NoError(t, m.InsertLeaf([]byte("node1")))
	require.NoError(t, m.InsertLeaf([]byte("node2")))
	require.NoError(t, m.InsertLeaf([]byte("node3")))
	require.NoError(t, m.InsertLeaf([]byte("node4")))
	require.NoError(t, m.InsertLeaf([]byte("node5")))
	require.NoError(t, m.InsertLeaf([]byte("node6")))

	hash12 := hashFn([]byte("node1"), []byte("node2"))
	hash34 := hashFn([]byte("node3"), []byte("node4"))
	hash56 := hashFn([]byte("node5"), []byte("node6"))
	hash66 := hashFn([]byte("node6"), []byte("node6"))
	hash1234 := hashFn(hash12[:], hash34[:])
	hash5666 := hashFn(hash56[:], hash66[:])
	hashRoot := hashFn(hash1234[:], hash5666[:])

	extraData := []byte("extra data")
	kvs, root, err = m.FinalizeWorkingTree(extraData)
	require.NoError(t, err)
	require.Equal(t, hashRoot[:], root)
	require.Len(t, kvs, 1)

	var info merkletypes.FinalizedTreeInfo
	require.NoError(t, json.Unmarshal(kvs[0].Value, &info))
	require.Equal(t, merkletypes.FinalizedTreeInfo{
		TreeIndex:      1,
		TreeHeight:     3,
		Root:           hashRoot[:],
		StartLeafIndex: 1,
		LeafCount:      6,
		ExtraData:      extraData,
	}, info)
}

func Test_GetProofs(t *testing.T) {
	tempDir := t.TempDir()
	db, err := db.NewDB(tempDir)
	require.NoError(t, err)

	hashFn := ophosttypes.GenerateNodeHash
	m, err := NewMerkle(db, hashFn)
	require.NoError(t, err)

	require.NoError(t, m.InitializeWorkingTree(1, 1))

	// insert 6 nodes
	require.NoError(t, m.InsertLeaf([]byte("node1")))
	require.NoError(t, m.InsertLeaf([]byte("node2")))
	require.NoError(t, m.InsertLeaf([]byte("node3")))
	require.NoError(t, m.InsertLeaf([]byte("node4")))
	require.NoError(t, m.InsertLeaf([]byte("node5")))
	require.NoError(t, m.InsertLeaf([]byte("node6")))

	hash12 := hashFn([]byte("node1"), []byte("node2"))
	hash34 := hashFn([]byte("node3"), []byte("node4"))
	hash56 := hashFn([]byte("node5"), []byte("node6"))
	hash66 := hashFn([]byte("node6"), []byte("node6"))
	hash1234 := hashFn(hash12[:], hash34[:])
	hash5666 := hashFn(hash56[:], hash66[:])
	hashRoot := hashFn(hash1234[:], hash5666[:])

	extraData := []byte("extra data")
	kvs, root, err := m.FinalizeWorkingTree(extraData)
	require.NoError(t, err)
	require.Equal(t, hashRoot[:], root)

	// store batch kvs to db
	require.NoError(t, db.RawBatchSet(kvs...))

	proofs, treeIndex, root_, extraData, err := m.GetProofs(1)
	require.NoError(t, err)
	require.Equal(t, uint64(1), treeIndex)
	require.Equal(t, root, root_)
	require.Equal(t, []byte("extra data"), extraData)
	require.Len(t, proofs, 3)
	require.Equal(t, []byte("node2"), proofs[0])
	require.Equal(t, hash34[:], proofs[1])
	require.Equal(t, hash5666[:], proofs[2])
}
