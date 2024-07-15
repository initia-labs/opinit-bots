package types

type QueryProofsResponse struct {
	BridgeId         uint64   `json:"bridge_id"`
	Sequence         uint64   `json:"sequence"`
	Version          []byte   `json:"version"`
	WithdrawalProofs [][]byte `json:"withdrawal_proofs"`
	OutputIndex      uint64   `json:"output_index"`
	StorageRoot      []byte   `json:"storage_root"`
	LatestBlockHash  []byte   `json:"latest_block_hash"`
	BlockNumber      uint64   `json:"block_number"`
}

type TreeExtraData struct {
	BlockNumber uint64 `json:"block_number"`
	BlockHash   []byte `json:"block_hash"`
}
