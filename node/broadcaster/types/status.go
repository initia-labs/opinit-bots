package types

type BroadcasterStatus struct {
	PendingTxs     int                        `json:"pending_txs"`
	AccountsStatus []BroadcasterAccountStatus `json:"accounts_status"`
}

type BroadcasterAccountStatus struct {
	Address  string `json:"address"`
	Sequence uint64 `json:"sequence"`
}
