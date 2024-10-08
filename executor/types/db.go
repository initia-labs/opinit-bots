package types

type WithdrawalData struct {
	Sequence       uint64 `json:"sequence"`
	From           string `json:"from"`
	To             string `json:"to"`
	Amount         uint64 `json:"amount"`
	BaseDenom      string `json:"base_denom"`
	WithdrawalHash []byte `json:"withdrawal_hash"`
}

type TreeExtraData struct {
	BlockNumber int64  `json:"block_number"`
	BlockHash   []byte `json:"block_hash"`
}
