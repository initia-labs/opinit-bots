package merkle

import (
	"encoding/json"
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
		LevelData:      make(map[uint8][]byte),
		Done:           false,
	}
}

func (m *Merkle) FinalizeWorkingTree() ([]types.KV, []byte, error) {
	if m.workingTree.LeafCount == 0 {
		return nil, merkletypes.EmptyRootHash[:], nil
	}
	kvs, err := m.fillRestLeaves()
	if err != nil {
		return nil, nil, err
	}

	treeRootHash := m.workingTree.LevelData[m.GetMaxLevel()]

	tree := merkletypes.FinalizedTreeInfo{
		TreeIndex:      m.workingTree.Index,
		Depth:          m.GetMaxLevel(),
		Root:           treeRootHash,
		StartLeafIndex: m.workingTree.StartLeafIndex,
		LeafCount:      m.workingTree.LeafCount,
	}

	data, err := json.Marshal(tree)
	if err != nil {
		return nil, nil, err
	}

	kvs = append(kvs, types.KV{
		Key:   merkletypes.PrefixedFinalizedTreeKey(m.workingTree.Index),
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
	data, err := json.Marshal(m.workingTree)
	if err != nil {
		return err
	}
	return m.db.Set(merkletypes.PrefixedWorkingTreeKey(version), data)
}

func (m *Merkle) GetKVWorkingTree() (types.KV, error) {
	data, err := json.Marshal(m.workingTree)
	if err != nil {
		return types.KV{}, err
	}
	return types.KV{
		Key:   merkletypes.WorkingTreeKey,
		Value: data,
	}, nil
}

func (m *Merkle) GetMaxLevel() uint8 {
	return uint8(bits.Len64(m.workingTree.LeafCount) - 1)
}

func (m *Merkle) GetWorkingTreeIndex() uint64 {
	return m.workingTree.Index
}

func (m *Merkle) saveNode(level uint8, levelIndex uint64, data []byte) error {
	return m.db.Set(merkletypes.PrefixedNodeKey(m.GetWorkingTreeIndex(), level, levelIndex), data)
}

func (m *Merkle) fillRestLeaves() ([]types.KV, error) {
	kvs := make([]types.KV, 0)
	leaf := m.workingTree.LevelData[0]

	numRestLeaves := 1<<(m.GetMaxLevel()+1) - m.workingTree.LeafCount
	for range numRestLeaves {
		err := m.InsertLeaf(leaf)
		if err != nil {
			return nil, err
		}
	}
	m.workingTree.LeafCount -= numRestLeaves
	m.workingTree.Done = true

	return kvs, nil
}

func (m *Merkle) InsertLeaf(data []byte) error {
	level := uint8(0)
	levelIndex := m.workingTree.LeafCount

	for {
		err := m.saveNode(level, levelIndex, data)
		if err != nil {
			return err
		}
		sibling := m.workingTree.LevelData[level]
		m.workingTree.LevelData[level] = data
		if levelIndex%2 == 0 {
			break
		}
		nodeHash := m.nodeGeneratorFn(sibling, data)
		data = nodeHash[:]
		levelIndex = levelIndex / 2
		level++
	}

	m.workingTree.LeafCount++
	return nil
}
