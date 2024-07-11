package types

import (
	"encoding/json"
	"fmt"
	"time"

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

func (p PendingTxInfo) String() string {
	tsStr := time.Unix(0, p.Timestamp).UTC().String()
	return fmt.Sprintf("Pending tx: %s, sequence: %d at height: %d, %s", p.TxHash, p.Sequence, p.ProcessedHeight, tsStr)
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

func (p ProcessedMsgs) String() string {
	msgStr := ""
	for _, msg := range p.Msgs {
		msgStr += msg.String() + ","
	}
	tsStr := time.Unix(0, p.Timestamp).UTC().String()
	return fmt.Sprintf("Pending msgs: %s at height: %d, %s", msgStr, p.Timestamp, tsStr)
}
