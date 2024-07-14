package types

type TreeInfo struct {
	Index          uint64           `json:"index"`
	LeafCount      uint64           `json:"leaf_count"`
	StartLeafIndex uint64           `json:"start_leaf_index"`
	HeightData     map[uint8][]byte `json:"height_data"`
	Done           bool             `json:"done"`
}

type FinalizedTreeInfo struct {
	TreeIndex  uint64 `json:"tree_index"`
	TreeHeight uint8  `json:"tree_height"`
	Root       []byte `json:"root"`
	// used to identify the first leaf index of the tree
	StartLeafIndex uint64 `json:"start_leaf_index"`
	LeafCount      uint64 `json:"leaf_count"`
	ExtraData      []byte `json:"extra_data,omitempty"`
}
