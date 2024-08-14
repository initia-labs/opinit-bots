package merkle

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/bits"

	merkletypes "github.com/initia-labs/opinit-bots-go/merkle/types"
	types "github.com/initia-labs/opinit-bots-go/types"
)

// NodeGeneratorFn is a function type that generates parent node from two child nodes.
//
// CONTRACT: It should generate return same result for same inputs even the order of inputs are swapped.
type NodeGeneratorFn func([]byte, []byte) [32]byte

// Merkle is a struct that manages the merkle tree which only holds the last sibling
// of each level(height) to minimize the memory usage.
type Merkle struct {
	db              types.DB
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

func NewMerkle(db types.DB, nodeGeneratorFn NodeGeneratorFn) (*Merkle, error) {
	err := validateNodeGeneratorFn(nodeGeneratorFn)
	if err != nil {
		return nil, err
	}

	return &Merkle{
		db:              db,
		nodeGeneratorFn: nodeGeneratorFn,
	}, nil
}

// InitializeWorkingTree resets the working tree with the given tree index and start leaf index.
func (m *Merkle) InitializeWorkingTree(treeIndex uint64, startLeafIndex uint64) error {
	if m.workingTree != nil && !m.workingTree.Done {
		return fmt.Errorf("failed to initialize working tree index: %d; working tree is not finalized", treeIndex)
	}

	if treeIndex < 1 || startLeafIndex < 1 {
		return fmt.Errorf("failed to initialize working tree index: %d, leaf: %d; invalid index", treeIndex, startLeafIndex)
	}

	m.workingTree = &merkletypes.TreeInfo{
		Index:          treeIndex,
		StartLeafIndex: startLeafIndex,
		LeafCount:      0,
		LastSiblings:   make(map[uint8][]byte),
		Done:           false,
	}

	return nil
}

// FinalizeWorkingTree finalizes the working tree and returns the finalized tree info.
func (m *Merkle) FinalizeWorkingTree(extraData []byte) ([]types.RawKV, []byte /* root */, error) {
	m.workingTree.Done = true
	if m.workingTree.LeafCount == 0 {
		return nil, merkletypes.EmptyRootHash[:], nil
	}

	err := m.fillLeaves()
	if err != nil {
		return nil, nil, err
	}

	treeRootHash := m.workingTree.LastSiblings[m.Height()]
	finalizedTreeInfo := merkletypes.FinalizedTreeInfo{
		TreeIndex:      m.workingTree.Index,
		TreeHeight:     m.Height(),
		Root:           treeRootHash,
		StartLeafIndex: m.workingTree.StartLeafIndex,
		LeafCount:      m.workingTree.LeafCount,
		ExtraData:      extraData,
	}

	data, err := json.Marshal(finalizedTreeInfo)
	if err != nil {
		return nil, nil, err
	}

	// Save the finalized tree info with the start leaf index as the key,
	// when we need to get the proofs for the leaf, we can get the tree info with the start leaf index.
	kvs := []types.RawKV{{
		Key:   m.db.PrefixedKey(finalizedTreeInfo.Key()),
		Value: data,
	}}

	return kvs, treeRootHash, err
}

// LoadWorkingTree loads the working tree from the database.
//
// It is used to load the working tree to handle the case where the bot is stopped.
func (m *Merkle) LoadWorkingTree(version uint64) error {
	data, err := m.db.Get(merkletypes.PrefixedWorkingTreeKey(version))
	if err != nil {
		return err
	}

	var workingTree merkletypes.TreeInfo
	err = json.Unmarshal(data, &workingTree)
	if err != nil {
		return err
	} else if workingTree.Done {
		nextTreeIndex := workingTree.Index + 1
		nextStartLeafIndex := workingTree.StartLeafIndex + workingTree.LeafCount
		return m.InitializeWorkingTree(nextTreeIndex, nextStartLeafIndex)
	}

	m.workingTree = &workingTree
	return nil
}

// SaveWorkingTree saves the working tree to the database.
//
// It is used to save the working tree to handle the case where the bot is stopped.
func (m *Merkle) SaveWorkingTree(version uint64) error {
	data, err := json.Marshal(&m.workingTree)
	if err != nil {
		return err
	}
	return m.db.Set(merkletypes.PrefixedWorkingTreeKey(version), data)
}

// Height returns the height of the working tree.
func (m *Merkle) Height() uint8 {
	if m.workingTree.LeafCount <= 1 {
		return uint8(m.workingTree.LeafCount)
	}

	return uint8(bits.Len64(m.workingTree.LeafCount - 1))
}

// GetWorkingTreeIndex returns the index of the working tree.
func (m *Merkle) GetWorkingTreeIndex() uint64 {
	return m.workingTree.Index
}

// GetWorkingTreeLeafCount returns the leaf count of the working tree.
func (m *Merkle) GetWorkingTreeLeafCount() uint64 {
	return m.workingTree.LeafCount
}

// GetStartLeafIndex returns the start leaf index of the working tree.
func (m *Merkle) GetStartLeafIndex() uint64 {
	return m.workingTree.StartLeafIndex
}

func (m *Merkle) saveNode(height uint8, localNodeIndex uint64, data []byte) error {
	return m.db.Set(merkletypes.PrefixedNodeKey(m.GetWorkingTreeIndex(), height, localNodeIndex), data)
}

func (m *Merkle) getNode(treeIndex uint64, height uint8, localNodeIndex uint64) ([]byte, error) {
	return m.db.Get(merkletypes.PrefixedNodeKey(treeIndex, height, localNodeIndex))
}

// fillLeaves fills the rest of the leaves with the last leaf.
func (m *Merkle) fillLeaves() error {
	numRestLeaves := 1<<(m.Height()) - m.workingTree.LeafCount
	if numRestLeaves == 0 {
		return nil
	}

	lastLeaf := m.workingTree.LastSiblings[0]
	for range numRestLeaves {
		if err := m.InsertLeaf(lastLeaf); err != nil {
			return err
		}
	}

	// leaf count increased with dummy values during the fill
	// process, so decrease it back to keep l2 withdrawal sequence mapping.
	m.workingTree.LeafCount -= numRestLeaves

	return nil
}

// InsertLeaf inserts a leaf to the working tree.
//
// It updates the last sibling of each level until the root.
func (m *Merkle) InsertLeaf(data []byte) error {
	height := uint8(0)
	localNodeIndex := m.workingTree.LeafCount

	for {
		// save the node with the given level and localLeafIndex
		err := m.saveNode(height, localNodeIndex, data)
		if err != nil {
			return err
		}

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

	return nil
}

// GetProofs returns the proofs for the leaf with the given index.
func (m *Merkle) GetProofs(leafIndex uint64) (proofs [][]byte, treeIndex uint64, rootData []byte, extraData []byte, err error) {
	_, value, err := m.db.SeekPrevInclusiveKey(merkletypes.FinalizedTreeKey, merkletypes.PrefixedFinalizedTreeKey(leafIndex))
	if err != nil {
		return nil, 0, nil, nil, err
	}

	var treeInfo merkletypes.FinalizedTreeInfo
	if err := json.Unmarshal(value, &treeInfo); err != nil {
		return nil, 0, nil, nil, err
	}

	// Check if the leaf index is in the tree
	if leafIndex < treeInfo.StartLeafIndex || leafIndex-treeInfo.StartLeafIndex >= treeInfo.LeafCount {
		return nil, 0, nil, nil, fmt.Errorf("leaf (`%d`) is not found in tree (`%d`)", leafIndex, treeInfo.TreeIndex)
	}

	height := uint8(0)
	localNodeIndex := leafIndex - treeInfo.StartLeafIndex
	for height < treeInfo.TreeHeight {
		siblingIndex := localNodeIndex ^ 1 // flip the last bit to find the sibling
		sibling, err := m.getNode(treeInfo.TreeIndex, height, siblingIndex)
		if err != nil {
			return nil, 0, nil, nil, err
		}

		// append the sibling to the proofs
		proofs = append(proofs, sibling)

		// update iteration variables
		height++
		localNodeIndex = localNodeIndex / 2
	}

	return proofs, treeInfo.TreeIndex, treeInfo.Root, treeInfo.ExtraData, nil
}
