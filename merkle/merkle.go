package merkle

import (
	"encoding/json"
	"errors"
	"math/bits"

	merkletypes "github.com/initia-labs/opinit-bots-go/merkle/types"
	types "github.com/initia-labs/opinit-bots-go/types"
)

type Merkle struct {
	db          types.DB
	workingTree merkletypes.TreeInfo

	nodeGeneratorFn func([]byte, []byte) [32]byte
}

func NewMerkle(db types.DB, nodeGeneratorFn func([]byte, []byte) [32]byte) *Merkle {
	return &Merkle{
		db:              db,
		nodeGeneratorFn: nodeGeneratorFn,
	}
}

func (m *Merkle) SetNewWorkingTree(treeIndex uint64, startLeafIndex uint64) {
	m.workingTree = merkletypes.TreeInfo{
		Index:          treeIndex,
		StartLeafIndex: startLeafIndex,
		LeafCount:      0,
		HeightData:     make(map[uint8][]byte),
		Done:           false,
	}
}

func (m *Merkle) FinalizeWorkingTree(extraData []byte) ([]types.KV, []byte, error) {
	m.workingTree.Done = true
	if m.workingTree.LeafCount == 0 {
		return nil, merkletypes.EmptyRootHash[:], nil
	}
	kvs, err := m.fillRestLeaves()
	if err != nil {
		return nil, nil, err
	}

	treeRootHash := m.workingTree.HeightData[m.Height()]
	tree := merkletypes.FinalizedTreeInfo{
		TreeIndex:      m.workingTree.Index,
		TreeHeight:     m.Height(),
		Root:           treeRootHash,
		StartLeafIndex: m.workingTree.StartLeafIndex,
		LeafCount:      m.workingTree.LeafCount,
		ExtraData:      extraData,
	}

	data, err := json.Marshal(tree)
	if err != nil {
		return nil, nil, err
	}

	kvs = append(kvs, types.KV{
		Key:   m.db.PrefixedKey(merkletypes.PrefixedFinalizedTreeKey(tree.StartLeafIndex)),
		Value: data,
	})

	return kvs, treeRootHash, err
}

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
		m.SetNewWorkingTree(workingTree.Index+1, workingTree.StartLeafIndex+workingTree.LeafCount)
		return nil
	}

	m.workingTree = workingTree
	return nil
}

func (m *Merkle) SaveWorkingTree(version uint64) error {
	data, err := json.Marshal(&m.workingTree)
	if err != nil {
		return err
	}
	return m.db.Set(merkletypes.PrefixedWorkingTreeKey(version), data)
}

func (m *Merkle) Height() uint8 {
	if m.workingTree.LeafCount <= 1 {
		return uint8(m.workingTree.LeafCount)
	}
	return uint8(bits.Len64(m.workingTree.LeafCount - 1))
}

func (m *Merkle) GetWorkingTreeIndex() uint64 {
	return m.workingTree.Index
}

func (m *Merkle) GetWorkingTreeLeafCount() uint64 {
	return m.workingTree.LeafCount
}

func (m *Merkle) saveNode(height uint8, heightIndex uint64, data []byte) error {
	return m.db.Set(merkletypes.PrefixedNodeKey(m.GetWorkingTreeIndex(), height, heightIndex), data)
}

func (m *Merkle) getNode(treeIndex uint64, height uint8, heightIndex uint64) ([]byte, error) {
	return m.db.Get(merkletypes.PrefixedNodeKey(treeIndex, height, heightIndex))
}

func (m *Merkle) fillRestLeaves() ([]types.KV, error) {
	kvs := make([]types.KV, 0)
	leaf := m.workingTree.HeightData[0]

	numRestLeaves := 1<<(m.Height()) - m.workingTree.LeafCount

	//nolint:typecheck
	for range numRestLeaves {
		err := m.InsertLeaf(leaf, true)
		if err != nil {
			return nil, err
		}
	}
	return kvs, nil
}

func (m *Merkle) InsertLeaf(data []byte, residue bool) error {
	height := uint8(0)
	heightIndex := m.workingTree.LeafCount

	for {
		err := m.saveNode(height, heightIndex, data)
		if err != nil {
			return err
		}
		sibling := m.workingTree.HeightData[height]
		m.workingTree.HeightData[height] = data
		if heightIndex%2 == 0 {
			break
		}
		nodeHash := m.nodeGeneratorFn(sibling, data)
		data = nodeHash[:]
		heightIndex = heightIndex / 2
		height++
	}

	if !residue {
		m.workingTree.LeafCount++
	}

	return nil
}

func (m *Merkle) GetProofs(leafIndex uint64) ([][]byte, uint64, []byte, []byte, error) {
	_, value, err := m.db.SeekPrevInclusiveKey(merkletypes.FinalizedTreeKey, merkletypes.PrefixedFinalizedTreeKey(leafIndex))
	if err != nil {
		return nil, 0, nil, nil, err
	}

	var treeInfo merkletypes.FinalizedTreeInfo
	err = json.Unmarshal(value, &treeInfo)
	if err != nil {
		return nil, 0, nil, nil, err
	}

	proofs := make([][]byte, 0)
	height := uint8(0)
	if leafIndex < treeInfo.StartLeafIndex || leafIndex-treeInfo.StartLeafIndex >= treeInfo.LeafCount {
		return nil, 0, nil, nil, errors.New("not found")
	}
	localIndex := leafIndex - treeInfo.StartLeafIndex

	for height < treeInfo.TreeHeight {
		sibling, err := m.getNode(treeInfo.TreeIndex, height, localIndex^1)
		if err != nil {
			return nil, 0, nil, nil, err
		}
		proofs = append(proofs, sibling)
		height++
		localIndex = localIndex / 2
	}
	return proofs, treeInfo.TreeIndex, treeInfo.Root, treeInfo.ExtraData, nil
}
