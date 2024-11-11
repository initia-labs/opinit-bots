package types

import "github.com/cosmos/cosmos-sdk/types"

type QueryWithdrawalResponse struct {
	// fields required to withdraw funds
	BridgeId         uint64     `json:"bridge_id"`
	OutputIndex      uint64     `json:"output_index"`
	WithdrawalProofs [][]byte   `json:"withdrawal_proofs"`
	From             string     `json:"from"`
	To               string     `json:"to"`
	Sequence         uint64     `json:"sequence"`
	Amount           types.Coin `json:"amount"`
	Version          []byte     `json:"version"`
	StorageRoot      []byte     `json:"storage_root"`
	LastBlockHash    []byte     `json:"last_block_hash"`

	// extra info
	// BlockNumber    int64  `json:"block_number"`
	// WithdrawalHash []byte `json:"withdrawal_hash"`
}
