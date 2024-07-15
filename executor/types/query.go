package types

type QueryProofsResponse struct {
	WithdrawalProofs [][]byte `json:"withdrawal_proofs"`
	OutputIndex      uint64   `json:"output_index"`
	StorageRoot      []byte   `json:"storage_root"`
	BlockNumber      uint64   `json:"block_number"`
	BlockHash        []byte   `json:"block_hash"`
}

type TreeExtraData struct {
	BlockNumber uint64 `json:"block_number"`
	BlockHash   []byte `json:"block_hash"`
}
