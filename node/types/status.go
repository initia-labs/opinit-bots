package types

type Status struct {
	LastProcessedBlockHeight uint64 `json:"last_processed_block_height"`
	PendingTxs               int    `json:"pending_txs,omitempty"`
	Sequence                 uint64 `json:"sequence,omitempty"`
}
