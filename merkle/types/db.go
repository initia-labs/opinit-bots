package types

type TreeInfo struct {
	Index     uint64           `json:"index"`
	LeafCount uint64           `json:"leaf_count"`
	LevelData map[uint8][]byte `json:"-"`
}

type FinalizedTree struct {
	Index uint64 `json:"index"`
	Depth uint8  `json:"depth"`
	Root  []byte `json:"root"`
}
