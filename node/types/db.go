package types

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type PendingTxInfo struct {
	ProcessedHeight int64  `json:"height"`
	Sequence        uint64 `json:"sequence"`
	Tx              []byte `json:"tx"`
	Save            bool   `json:"save"`
}

func (p PendingTxInfo) Marshal() ([]byte, error) {
	return json.Marshal(&p)
}

func (p *PendingTxInfo) Unmarshal(data []byte) error {
	return json.Unmarshal(data, p)
}

type ProcessedData []ProcessedMsgs

type ProcessedMsgs struct {
	Msgs []sdk.Msg `json:"msgs"`
	Save bool      `json:"save"`
}

func (p ProcessedMsgs) Marshal() ([]byte, error) {
	return json.Marshal(&p)
}

func (p *ProcessedMsgs) Unmarshal(data []byte) error {
	return json.Unmarshal(data, p)
}
