package types

type TreeInfo struct {
	Index          uint64           `json:"index"`
	LeafCount      uint64           `json:"leaf_count"`
	StartLeafIndex uint64           `json:"start_leaf_index"`
	LevelData      map[uint8][]byte `json:"-"`
}

type FinalizedTreeInfo struct {
	TreeIndex uint64 `json:"tree_index"`
	Depth     uint8  `json:"depth"`
	Root      []byte `json:"root"`
	// used to identify the first leaf index of the tree
	StartLeafIndex uint64 `json:"start_leaf_index"`
}
