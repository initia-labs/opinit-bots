package merkle

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/bits"

	merkletypes "github.com/initia-labs/opinit-bots/merkle/types"
	types "github.com/initia-labs/opinit-bots/types"
)

// NodeGeneratorFn is a function type that generates parent node from two child nodes.
//
// CONTRACT: It should generate return same result for same inputs even the order of inputs are swapped.
type NodeGeneratorFn func([]byte, []byte) [32]byte

// Merkle is a struct that manages the merkle tree which only holds the last sibling
// of each level(height) to minimize the memory usage.
type Merkle struct {
	workingTree     *merkletypes.TreeInfo
	nodeGeneratorFn NodeGeneratorFn
}

// Check if the node generator function is commutative
func validateNodeGeneratorFn(fn NodeGeneratorFn) error {
	randInput1 := make([]byte, 32)
	randInput2 := make([]byte, 32)
	_, err := rand.Read(randInput1)
	if err != nil {
		return err
	}
	_, err = rand.Read(randInput2)
	if err != nil {
		return err
	}

	node1 := fn(randInput1, randInput2)
	node2 := fn(randInput2, randInput1)

	if node1 != node2 {
		return errors.New("node generator function is not commutative")
	}

	return nil
}

func NewMerkle(nodeGeneratorFn NodeGeneratorFn) (*Merkle, error) {
	err := validateNodeGeneratorFn(nodeGeneratorFn)
	if err != nil {
		return nil, err
	}

	return &Merkle{
		nodeGeneratorFn: nodeGeneratorFn,
	}, nil
}

// InitializeWorkingTree resets the working tree with the given tree index and start leaf index.
func (m *Merkle) InitializeWorkingTree(version uint64, treeIndex uint64, startLeafIndex uint64) error {
	if treeIndex < 1 || startLeafIndex < 1 {
		return fmt.Errorf("failed to initialize working tree index: %d, leaf: %d; invalid index", treeIndex, startLeafIndex)
	}

	m.workingTree = &merkletypes.TreeInfo{
		Version:        version,
		Index:          treeIndex,
		StartLeafIndex: startLeafIndex,
		LeafCount:      0,
		LastSiblings:   make(map[uint8][]byte),
		Done:           false,
	}
	return nil
}

// FinalizeWorkingTree finalizes the working tree and returns the finalized tree info.
func (m *Merkle) FinalizeWorkingTree(extraData []byte) (*merkletypes.FinalizedTreeInfo, []merkletypes.Node, []byte /* root */, error) {
	if m.workingTree == nil {
		return nil, nil, nil, errors.New("working tree is not initialized")
	}
	m.workingTree.Done = true
	if m.workingTree.LeafCount == 0 {
		return nil, nil, merkletypes.EmptyRootHash[:], nil
	}

	newNodes, err := m.fillLeaves()
	if err != nil {
		return nil, nil, nil, err
	}

	height, err := m.Height()
	if err != nil {
		return nil, nil, nil, err
	}

	treeRootHash := m.workingTree.LastSiblings[height]
	finalizedTreeInfo := &merkletypes.FinalizedTreeInfo{
		TreeIndex:      m.workingTree.Index,
		TreeHeight:     height,
		Root:           treeRootHash,
		StartLeafIndex: m.workingTree.StartLeafIndex,
		LeafCount:      m.workingTree.LeafCount,
		ExtraData:      extraData,
	}

	return finalizedTreeInfo, newNodes, treeRootHash, nil
}

// LoadWorkingTree loads the working tree from the database.
//
// It is used to load the working tree to handle the case where the bot is stopped.
func (m *Merkle) PrepareWorkingTree(lastWorkingTree merkletypes.TreeInfo) error {
	m.workingTree = &lastWorkingTree
	m.workingTree.Version++

	if m.workingTree.Done {
		nextTreeIndex := m.workingTree.Index + 1
		nextStartLeafIndex := m.workingTree.StartLeafIndex + m.workingTree.LeafCount
		return m.InitializeWorkingTree(m.workingTree.Version, nextTreeIndex, nextStartLeafIndex)
	}
	return nil
}

// Height returns the height of the working tree.
//
// Example:
// - For 7 leaves, the height is 3.
// - For 8 leaves, the height is 3.
// - For 9 leaves, the height is 4.
// - For 16 leaves, the height is 4.
func (m *Merkle) Height() (uint8, error) {
	if m.workingTree == nil {
		return 0, errors.New("working tree is not initialized")
	}

	leafCount := m.workingTree.LeafCount
	if leafCount <= 1 {
		return uint8(leafCount), nil
	}
	return types.MustIntToUint8(bits.Len64(leafCount - 1)), nil
}

// WorkingTree returns the working tree.
func (m *Merkle) WorkingTree() (merkletypes.TreeInfo, error) {
	if m.workingTree == nil {
		return merkletypes.TreeInfo{}, errors.New("working tree is not initialized")
	}
	return *m.workingTree, nil
}

// fillLeaves fills the rest of the leaves with the last leaf.
func (m *Merkle) fillLeaves() ([]merkletypes.Node, error) {
	if m.workingTree == nil {
		return nil, errors.New("working tree is not initialized")
	}
	height, err := m.Height()
	if err != nil {
		return nil, err
	}
	numRestLeaves := 1<<height - m.workingTree.LeafCount
	if numRestLeaves == 0 {
		return nil, nil
	}

	newNodes := make([]merkletypes.Node, 0)
	lastLeaf := m.workingTree.LastSiblings[0]
	for i := uint64(0); i < numRestLeaves; i++ {
		nodes, err := m.InsertLeaf(lastLeaf)
		if err != nil {
			return nil, err
		}
		newNodes = append(newNodes, nodes...)
	}

	// leaf count increased with dummy values during the fill
	// process, so decrease it back to keep l2 withdrawal sequence mapping.
	m.workingTree.LeafCount -= numRestLeaves
	return newNodes, nil
}

// InsertLeaf inserts a leaf to the working tree.
//
// It updates the last sibling of each level until the root.
func (m *Merkle) InsertLeaf(data []byte) ([]merkletypes.Node, error) {
	if m.workingTree == nil {
		return nil, errors.New("working tree is not initialized")
	}
	height := uint8(0)
	localNodeIndex := m.workingTree.LeafCount

	newNodes := make([]merkletypes.Node, 0)
	for {
		newNodes = append(newNodes, merkletypes.Node{
			TreeIndex:      m.workingTree.Index,
			Height:         height,
			LocalNodeIndex: localNodeIndex,
			Data:           data,
		})

		sibling := m.workingTree.LastSiblings[height]
		m.workingTree.LastSiblings[height] = data
		if localNodeIndex%2 == 0 {
			break
		}

		// if localLeafIndex is odd, calculate parent node
		nodeHash := m.nodeGeneratorFn(sibling, data)
		data = nodeHash[:]
		localNodeIndex = localNodeIndex / 2
		height++
	}

	m.workingTree.LeafCount++
	return newNodes, nil
}
