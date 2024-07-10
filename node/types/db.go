package types

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type PendingTxInfo struct {
	ProcessedHeight int64  `json:"height"`
	Sequence        uint64 `json:"sequence"`
	Tx              []byte `json:"tx"`
	TxHash          string `json:"tx_hash"`
	Timestamp       int64  `json:"timestamp"`
	Save            bool   `json:"save"`
}

func (p PendingTxInfo) Marshal() ([]byte, error) {
	return json.Marshal(&p)
}

func (p *PendingTxInfo) Unmarshal(data []byte) error {
	return json.Unmarshal(data, p)
}

type ProcessedMsgs struct {
	Msgs      []sdk.Msg `json:"msgs"`
	Timestamp int64     `json:"timestamp"`
	Save      bool      `json:"save"`
}

func (p ProcessedMsgs) Marshal() ([]byte, error) {
	return json.Marshal(&p)
}

func (p *ProcessedMsgs) Unmarshal(data []byte) error {
	return json.Unmarshal(data, p)
}
