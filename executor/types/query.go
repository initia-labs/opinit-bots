package types

import "github.com/cosmos/cosmos-sdk/types"

type QueryWithdrawalResponse struct {
	// fields required to withdraw funds
	Sequence         uint64     `json:"sequence"`
	To               string     `json:"to"`
	From             string     `json:"from"`
	Amount           types.Coin `json:"amount"`
	OutputIndex      uint64     `json:"output_index"`
	BridgeId         uint64     `json:"bridge_id"`
	WithdrawalProofs [][]byte   `json:"withdrawal_proofs"`
	Version          []byte     `json:"version"`
	StorageRoot      []byte     `json:"storage_root"`
	LastBlockHash    []byte     `json:"last_block_hash"`

	// extra info
	// BlockNumber    int64  `json:"block_number"`
	// WithdrawalHash []byte `json:"withdrawal_hash"`
}

type QueryWithdrawalsResponse struct {
	Withdrawals []QueryWithdrawalResponse `json:"withdrawals"`
	Next        uint64                    `json:"next"`
	Total       uint64                    `json:"total"`
}
