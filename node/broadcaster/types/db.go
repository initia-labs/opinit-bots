package types

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type PendingTxInfo struct {
	ProcessedHeight uint64 `json:"height"`
	Sequence        uint64 `json:"sequence"`
	Tx              []byte `json:"tx"`
	TxHash          string `json:"tx_hash"`
	Timestamp       int64  `json:"timestamp"`

	// Save is true if the pending tx should be saved until processed.
	// Save is false if the pending tx can be discarded even if it is not processed
	// like oracle tx.
	Save bool `json:"save"`
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

	// Save is true if the processed msgs should be saved until processed.
	// Save is false if the processed msgs can be discarded even if they are not processed
	// like oracle msgs.
	Save bool `json:"save"`
}

// processedMsgsJSON is a helper struct to JSON encode ProcessedMsgs
type processedMsgsJSON struct {
	Msgs      []string `json:"msgs"`
	Timestamp int64    `json:"timestamp"`
	Save      bool     `json:"save"`
}

func (p ProcessedMsgs) MarshalInterfaceJSON(cdc codec.Codec) ([]byte, error) {
	pms := processedMsgsJSON{
		Msgs:      make([]string, len(p.Msgs)),
		Timestamp: p.Timestamp,
		Save:      p.Save,
	}

	for i, msg := range p.Msgs {
		bz, err := cdc.MarshalInterfaceJSON(msg)
		if err != nil {
			return nil, err
		}

		pms.Msgs[i] = string(bz)
	}

	return json.Marshal(&pms)
}

func (p *ProcessedMsgs) UnmarshalInterfaceJSON(cdc codec.Codec, data []byte) error {
	var pms processedMsgsJSON
	if err := json.Unmarshal(data, &pms); err != nil {
		return err
	}

	p.Timestamp = pms.Timestamp
	p.Save = pms.Save

	p.Msgs = make([]sdk.Msg, len(pms.Msgs))
	for i, msgStr := range pms.Msgs {
		if err := cdc.UnmarshalInterfaceJSON([]byte(msgStr), &p.Msgs[i]); err != nil {
			return err
		}
	}

	return nil
}

func (p ProcessedMsgs) String() string {
	msgStr := ""
	for _, msg := range p.Msgs {
		msgStr += sdk.MsgTypeURL(msg) + ","
	}
	tsStr := time.Unix(0, p.Timestamp).UTC().String()
	return fmt.Sprintf("Pending msgs: %s at %s", msgStr, tsStr)
}
