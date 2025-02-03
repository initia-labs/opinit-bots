package merkle

import (
	"testing"

	"golang.org/x/crypto/sha3"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	merkletypes "github.com/initia-labs/opinit-bots/merkle/types"
	"github.com/stretchr/testify/require"
)

func TestValidateNodeGeneratorFn(t *testing.T) {
	fnNonCommutable := func(a, b []byte) [32]byte {
		return sha3.Sum256(append(a, b...))
	}

	fnCommutable := ophosttypes.GenerateNodeHash
	require.Error(t, validateNodeGeneratorFn(fnNonCommutable))
	require.NoError(t, validateNodeGeneratorFn(fnCommutable))
}

func TestInitializeWorkingTree(t *testing.T) {
	hashFn := ophosttypes.GenerateNodeHash
	m, err := NewMerkle(hashFn)
	require.NoError(t, err)

	cases := []struct {
		title          string
		version        uint64
		treeIndex      uint64
		startLeafIndex uint64
		expected       bool
	}{
		{"simple treeIndex, startLeafIndex", 10, 5, 10, true},
		{"zero treeIndex", 10, 0, 3, false},
		{"zero startLeafIndex", 10, 3, 0, false},
	}

	for _, tc := range cases {
		t.Run(tc.title, func(t *testing.T) {
			err := m.InitializeWorkingTree(tc.version, tc.treeIndex, tc.startLeafIndex)
			if tc.expected {
				require.NoError(t, err)
				require.Equal(t, &merkletypes.TreeInfo{
					Version:        tc.version,
					Index:          tc.treeIndex,
					StartLeafIndex: tc.startLeafIndex,
					LeafCount:      0,
					LastSiblings:   make(map[uint8][]byte),
					Done:           false,
				}, m.workingTree)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestHeight(t *testing.T) {
	hashFn := ophosttypes.GenerateNodeHash
	m, err := NewMerkle(hashFn)
	require.NoError(t, err)

	cases := []struct {
		title    string
		tree     *merkletypes.TreeInfo
		height   uint8
		expected bool
	}{
		{"0 leaf", &merkletypes.TreeInfo{LeafCount: 0}, 0, true},
		{"1 leaves", &merkletypes.TreeInfo{LeafCount: 1}, 1, true},
		{"2 leaves", &merkletypes.TreeInfo{LeafCount: 2}, 1, true},
		{"5 leaves", &merkletypes.TreeInfo{LeafCount: 5}, 3, true},
		{"1048576 leaves", &merkletypes.TreeInfo{LeafCount: 1048576}, 20, true},
		{"no tree", nil, 0, false},
	}

	for _, tc := range cases {
		t.Run(tc.title, func(t *testing.T) {
			m.workingTree = tc.tree
			height, err := m.Height()
			if tc.expected {
				require.NoError(t, err)
				require.Equal(t, tc.height, height)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestWorkingTree(t *testing.T) {
	hashFn := ophosttypes.GenerateNodeHash
	m, err := NewMerkle(hashFn)
	require.NoError(t, err)

	_, err = m.WorkingTree()
	require.Error(t, err)

	err = m.InitializeWorkingTree(10, 1, 1)
	require.NoError(t, err)

	_, err = m.WorkingTree()
	require.NoError(t, err)
}

func TestFillLeaves(t *testing.T) {
	hashFn := ophosttypes.GenerateNodeHash
	m, err := NewMerkle(hashFn)
	require.NoError(t, err)

	_, err = m.fillLeaves()
	require.Error(t, err)

	nodeData := []byte("node")

	hash12 := hashFn(nodeData, nodeData)
	hash56 := hashFn([]byte("node"), []byte("node"))
	hash66 := hashFn([]byte("node"), []byte("node"))
	hash1234 := hashFn(hash12[:], hash12[:])
	hash5666 := hashFn(hash56[:], hash66[:])
	hashRoot := hashFn(hash1234[:], hash5666[:])

	cases := []struct {
		title    string
		leaves   uint64
		nodes    []merkletypes.Node
		expected bool
	}{
		{"0 leaf", 0, []merkletypes.Node{
			{
				TreeIndex:      1,
				Height:         0,
				LocalNodeIndex: 0,
				Data:           []byte(nil),
			},
		}, true},
		{"1 leaves", 1, []merkletypes.Node{
			{
				TreeIndex:      1,
				Height:         0,
				LocalNodeIndex: 1,
				Data:           []byte("node"),
			},
			{
				TreeIndex:      1,
				Height:         1,
				LocalNodeIndex: 0,
				Data:           hash12[:],
			},
		}, true},
		{"2 leaves", 2, nil, true},
		{"5 leaves", 5, []merkletypes.Node{
			{
				TreeIndex:      1,
				Height:         0,
				LocalNodeIndex: 5,
				Data:           []byte("node"),
			},
			{
				TreeIndex:      1,
				Height:         1,
				LocalNodeIndex: 2,
				Data:           hash56[:],
			},
			{
				TreeIndex:      1,
				Height:         0,
				LocalNodeIndex: 6,
				Data:           []byte("node"),
			},
			{
				TreeIndex:      1,
				Height:         0,
				LocalNodeIndex: 7,
				Data:           []byte("node"),
			},
			{
				TreeIndex:      1,
				Height:         1,
				LocalNodeIndex: 3,
				Data:           hash66[:],
			},
			{
				TreeIndex:      1,
				Height:         2,
				LocalNodeIndex: 1,
				Data:           hash5666[:],
			},
			{
				TreeIndex:      1,
				Height:         3,
				LocalNodeIndex: 0,
				Data:           hashRoot[:],
			},
		}, true},
		{"1048576 leaves", 1048576, nil, true},
	}

	for _, tc := range cases {
		t.Run(tc.title, func(t *testing.T) {
			err = m.InitializeWorkingTree(10, 1, 1)
			require.NoError(t, err)

			for i := uint64(0); i < tc.leaves; i++ {
				_, err := m.InsertLeaf([]byte("node"))
				require.NoError(t, err)
			}

			nodes, err := m.fillLeaves()

			if tc.expected {
				require.NoError(t, err)
				require.Equal(t, tc.nodes, nodes)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestInsertLeaf(t *testing.T) {
	hashFn := ophosttypes.GenerateNodeHash
	m, err := NewMerkle(hashFn)
	require.NoError(t, err)

	require.NoError(t, m.InitializeWorkingTree(10, 1, 1))

	// empty tree
	require.Len(t, m.workingTree.LastSiblings, 0)

	// 1 node
	nodes, err := m.InsertLeaf([]byte("node1"))
	require.NoError(t, err)
	require.Len(t, m.workingTree.LastSiblings, 1)
	require.Equal(t, []byte("node1"), m.workingTree.LastSiblings[0])
	require.Len(t, nodes, 1)
	require.Equal(t, merkletypes.Node{
		TreeIndex:      1,
		Height:         0,
		LocalNodeIndex: 0,
		Data:           []byte("node1"),
	}, nodes[0])

	// 2 nodes
	hash12 := hashFn([]byte("node1"), []byte("node2"))
	nodes, err = m.InsertLeaf([]byte("node2"))
	require.NoError(t, err)
	require.Len(t, m.workingTree.LastSiblings, 2)
	require.Equal(t, []byte("node2"), m.workingTree.LastSiblings[0])
	require.Equal(t, hash12[:], m.workingTree.LastSiblings[1])
	require.Len(t, nodes, 2)
	require.Equal(t, merkletypes.Node{
		TreeIndex:      1,
		Height:         0,
		LocalNodeIndex: 1,
		Data:           []byte("node2"),
	}, nodes[0])
	require.Equal(t, merkletypes.Node{
		TreeIndex:      1,
		Height:         1,
		LocalNodeIndex: 0,
		Data:           hash12[:],
	}, nodes[1])

	// 3 nodes
	nodes, err = m.InsertLeaf([]byte("node3"))
	require.NoError(t, err)
	require.Len(t, m.workingTree.LastSiblings, 2)
	require.Equal(t, []byte("node3"), m.workingTree.LastSiblings[0])
	require.Equal(t, hash12[:], m.workingTree.LastSiblings[1])
	require.Len(t, nodes, 1)
	require.Equal(t, merkletypes.Node{
		TreeIndex:      1,
		Height:         0,
		LocalNodeIndex: 2,
		Data:           []byte("node3"),
	}, nodes[0])

	// 4 nodes
	hash34 := hashFn([]byte("node3"), []byte("node4"))
	hash1234 := hashFn(hash12[:], hash34[:])
	nodes, err = m.InsertLeaf([]byte("node4"))
	require.NoError(t, err)
	require.Len(t, m.workingTree.LastSiblings, 3)
	require.Equal(t, []byte("node4"), m.workingTree.LastSiblings[0])
	require.Equal(t, hash34[:], m.workingTree.LastSiblings[1])
	require.Equal(t, hash1234[:], m.workingTree.LastSiblings[2])
	require.Len(t, nodes, 3)
	require.Equal(t, merkletypes.Node{
		TreeIndex:      1,
		Height:         0,
		LocalNodeIndex: 3,
		Data:           []byte("node4"),
	}, nodes[0])
	require.Equal(t, merkletypes.Node{
		TreeIndex:      1,
		Height:         1,
		LocalNodeIndex: 1,
		Data:           hash34[:],
	}, nodes[1])
	require.Equal(t, merkletypes.Node{
		TreeIndex:      1,
		Height:         2,
		LocalNodeIndex: 0,
		Data:           hash1234[:],
	}, nodes[2])

	// 5 nodes
	nodes, err = m.InsertLeaf([]byte("node5"))
	require.NoError(t, err)
	require.Len(t, m.workingTree.LastSiblings, 3)
	require.Equal(t, []byte("node5"), m.workingTree.LastSiblings[0])
	require.Equal(t, hash34[:], m.workingTree.LastSiblings[1])
	require.Equal(t, hash1234[:], m.workingTree.LastSiblings[2])
	require.Len(t, nodes, 1)
	require.Equal(t, merkletypes.Node{
		TreeIndex:      1,
		Height:         0,
		LocalNodeIndex: 4,
		Data:           []byte("node5"),
	}, nodes[0])

	// 6 nodes
	hash56 := hashFn([]byte("node5"), []byte("node6"))
	nodes, err = m.InsertLeaf([]byte("node6"))
	require.NoError(t, err)
	require.Len(t, m.workingTree.LastSiblings, 3)
	require.Equal(t, []byte("node6"), m.workingTree.LastSiblings[0])
	require.Equal(t, hash56[:], m.workingTree.LastSiblings[1])
	require.Equal(t, hash1234[:], m.workingTree.LastSiblings[2])
	require.Len(t, nodes, 2)
	require.Equal(t, merkletypes.Node{
		TreeIndex:      1,
		Height:         0,
		LocalNodeIndex: 5,
		Data:           []byte("node6"),
	}, nodes[0])
	require.Equal(t, merkletypes.Node{
		TreeIndex:      1,
		Height:         1,
		LocalNodeIndex: 2,
		Data:           hash56[:],
	}, nodes[1])
}

func TestFinalizeWorkingTree(t *testing.T) {
	hashFn := ophosttypes.GenerateNodeHash
	m, err := NewMerkle(hashFn)
	require.NoError(t, err)

	require.NoError(t, m.InitializeWorkingTree(10, 1, 1))

	// empty tree
	finalizedTree, newNodes, root, err := m.FinalizeWorkingTree(nil)
	require.NoError(t, err)
	require.Equal(t, merkletypes.EmptyRootHash[:], root)
	require.Nil(t, finalizedTree)
	require.Len(t, newNodes, 0)

	// insert 6 nodes
	_, err = m.InsertLeaf([]byte("node1"))
	require.NoError(t, err)
	_, err = m.InsertLeaf([]byte("node2"))
	require.NoError(t, err)
	_, err = m.InsertLeaf([]byte("node3"))
	require.NoError(t, err)
	_, err = m.InsertLeaf([]byte("node4"))
	require.NoError(t, err)
	_, err = m.InsertLeaf([]byte("node5"))
	require.NoError(t, err)
	_, err = m.InsertLeaf([]byte("node6"))
	require.NoError(t, err)

	hash12 := hashFn([]byte("node1"), []byte("node2"))
	hash34 := hashFn([]byte("node3"), []byte("node4"))
	hash56 := hashFn([]byte("node5"), []byte("node6"))
	hash66 := hashFn([]byte("node6"), []byte("node6"))
	hash1234 := hashFn(hash12[:], hash34[:])
	hash5666 := hashFn(hash56[:], hash66[:])
	hashRoot := hashFn(hash1234[:], hash5666[:])

	extraData := []byte("extra data")
	finalizedTree, newNodes, root, err = m.FinalizeWorkingTree(extraData)
	require.NoError(t, err)
	require.Equal(t, hashRoot[:], root)
	// 7, 8, 78, 5678, 12345678
	require.Len(t, newNodes, 5)

	require.Equal(t, merkletypes.FinalizedTreeInfo{
		TreeIndex:      1,
		TreeHeight:     3,
		Root:           hashRoot[:],
		StartLeafIndex: 1,
		LeafCount:      6,
		ExtraData:      extraData,
	}, *finalizedTree)
}
