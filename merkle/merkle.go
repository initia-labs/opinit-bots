package merkle

import (
	"encoding/json"
	"math/bits"

	dbtypes "github.com/initia-labs/opinit-bots-go/db/types"
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

func (m *Merkle) NextWorkingTree() error {
	m.workingTree.Index++
	m.workingTree.StartLeafIndex += m.workingTree.LeafCount
	m.workingTree.LeafCount = 0
	m.workingTree.LevelData = make(map[uint8][]byte)
	return nil
}

func (m *Merkle) FinishWorkingTree() ([]types.KV, []byte, error) {
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

func (m *Merkle) LoadWorkingTree(workingIndex uint64) error {
	data, err := m.db.Get(merkletypes.WorkingTreeKey)
	if err == dbtypes.ErrNotFound {
		m.workingTree = merkletypes.TreeInfo{
			Index:     workingIndex,
			LeafCount: 0,
			LevelData: make(map[uint8][]byte),
		}
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &m.workingTree)
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

func (m *Merkle) getLevelData(level uint8, levelIndex uint64) ([]byte, error) {
	if m.workingTree.LevelData[level] != nil {
		return m.workingTree.LevelData[level], nil
	}

	data, err := m.db.Get(merkletypes.PrefixedNodeKey(m.GetWorkingTreeIndex(), level, levelIndex))
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (m *Merkle) fillRestLeaves() ([]types.KV, error) {
	kvs := make([]types.KV, 0)
	leaf := m.workingTree.LevelData[0]

	numRestLeaves := 1<<(m.GetMaxLevel()+1) - m.workingTree.LeafCount
	for range numRestLeaves {
		subKVs, err := m.insertLeaf(leaf)
		if err != nil {
			return nil, err
		}
		kvs = append(kvs, subKVs...)
	}
	kv, err := m.GetKVWorkingTree()
	if err != nil {
		return nil, err
	}
	kvs = append(kvs, kv)
	return kvs, nil
}

func (m *Merkle) InsertLeaves(leaves [][]byte) ([]types.KV, error) {
	kvs := make([]types.KV, 0)
	for _, leaf := range leaves {
		subKVs, err := m.insertLeaf(leaf)
		if err != nil {
			return nil, err
		}
		kvs = append(kvs, subKVs...)
	}
	kv, err := m.GetKVWorkingTree()
	if err != nil {
		return nil, err
	}
	kvs = append(kvs, kv)
	return kvs, nil
}

func (m *Merkle) insertLeaf(data []byte) ([]types.KV, error) {
	kvs := make([]types.KV, 0)
	level := uint8(0)
	levelIndex := m.workingTree.LeafCount

	for {
		kvs = append(kvs, types.KV{
			Key:   merkletypes.PrefixedNodeKey(m.GetWorkingTreeIndex(), level, levelIndex),
			Value: data,
		})
		m.workingTree.LevelData[level] = data

		if levelIndex%2 == 0 {
			break
		}

		sibling, err := m.getLevelData(level, levelIndex-1)
		if err != nil {
			return nil, err
		}
		nodeHash := m.nodeGeneratorFn(sibling, data)
		data = nodeHash[:]
		levelIndex = levelIndex / 2
		level++
	}

	m.workingTree.LeafCount++
	return kvs, nil
}

func (m *Merkle) DropLevelData() {
	m.workingTree.LevelData = make(map[uint8][]byte)
}
