package types

type QueryWithdrawalResponse struct {
	// fields required to withdraw funds
	BridgeId         uint64   `json:"bridge_id"`
	OutputIndex      uint64   `json:"output_index"`
	WithdrawalProofs [][]byte `json:"withdrawal_proofs"`
	Sender           string   `json:"sender"`
	Sequence         uint64   `json:"sequence"`
	Amount           string   `json:"amount"`
	Version          []byte   `json:"version"`
	StorageRoot      []byte   `json:"storage_root"`
	LatestBlockHash  []byte   `json:"latest_block_hash"`

	// extra info
	BlockNumber    uint64 `json:"block_number"`
	Receiver       string `json:"receiver"`
	WithdrawalHash []byte `json:"withdrawal_hash"`
}
