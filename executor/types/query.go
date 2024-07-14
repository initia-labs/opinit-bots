package types

type QueryProofsResponse struct {
	WithdrawalProofs [][]byte `json:"withdrawal_proofs"`
	OutputIndex      uint64   `json:"output_index"`
	StorageRoot      []byte   `json:"storage_root"`
	LatestBlockHash  []byte   `json:"latest_block_hash"`
}
