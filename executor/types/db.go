package types

import "encoding/json"

type Withdrawals struct {
	Height      uint64       `json:"height"`
	Withdrawals []Withdrawal `json:"withdrawals"`
}

type Withdrawal struct {
	Sequence       uint64   `json:"sequence"`
	WithdrawalHash [32]byte `json:"withdrawal_hash"`
}

func (w Withdrawals) Marshal() ([]byte, error) {
	return json.Marshal(&w)
}

func (w *Withdrawals) Unmarshal(data []byte) error {
	return json.Unmarshal(data, w)
}
