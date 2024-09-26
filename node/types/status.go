package types

type BroadcasterStatus struct {
	PendingTxs int    `json:"pending_txs"`
	Sequence   uint64 `json:"sequence"`
}

type Status struct {
	LastBlockHeight int64              `json:"last_block_height,omitempty"`
	Broadcaster     *BroadcasterStatus `json:"broadcaster,omitempty"`
}
