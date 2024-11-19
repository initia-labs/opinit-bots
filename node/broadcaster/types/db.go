package types

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/initia-labs/opinit-bots/types"
)

type PendingTxInfo struct {
	Sender          string   `json:"sender"`
	ProcessedHeight int64    `json:"height"`
	Sequence        uint64   `json:"sequence"`
	Tx              []byte   `json:"tx"`
	TxHash          string   `json:"tx_hash"`
	Timestamp       int64    `json:"timestamp"`
	MsgTypes        []string `json:"msg_types"`

	// Save is true if the pending tx should be saved until processed.
	// Save is false if the pending tx can be discarded even if it is not processed
	// like oracle tx.
	Save bool `json:"save"`
}

func (p PendingTxInfo) Key() []byte {
	return prefixedPendingTx(p.Sequence)
}

func (p PendingTxInfo) Value() ([]byte, error) {
	return p.Marshal()
}

func (p PendingTxInfo) Marshal() ([]byte, error) {
	return json.Marshal(&p)
}

func (p *PendingTxInfo) Unmarshal(data []byte) error {
	return json.Unmarshal(data, p)
}

func (p PendingTxInfo) String() string {
	tsStr := time.Unix(0, p.Timestamp).UTC().String()
	return fmt.Sprintf("Pending tx: %s, sender: %s, msgs: %s, sequence: %d at height: %d, %s", p.TxHash, p.Sender, strings.Join(p.MsgTypes, ","), p.Sequence, p.ProcessedHeight, tsStr)
}

type ProcessedMsgs struct {
	Sender    string    `json:"sender"`
	Msgs      []sdk.Msg `json:"msgs"`
	Timestamp int64     `json:"timestamp"`

	// Save is true if the processed msgs should be saved until processed.
	// Save is false if the processed msgs can be discarded even if they are not processed
	// like oracle msgs.
	Save bool `json:"save"`
}

// processedMsgsJSON is a helper struct to JSON encode ProcessedMsgs
type processedMsgsJSON struct {
	Sender    string   `json:"sender"`
	Msgs      []string `json:"msgs"`
	Timestamp int64    `json:"timestamp"`
	Save      bool     `json:"save"`
}

func (p ProcessedMsgs) Key() []byte {
	return prefixedProcessedMsgs(types.MustInt64ToUint64(p.Timestamp))
}

func (p ProcessedMsgs) Value(cdc codec.Codec) ([]byte, error) {
	return p.MarshalInterfaceJSON(cdc)
}

func (p ProcessedMsgs) MarshalInterfaceJSON(cdc codec.Codec) ([]byte, error) {
	pms := processedMsgsJSON{
		Sender:    p.Sender,
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

	p.Sender = pms.Sender
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
	tsStr := time.Unix(0, p.Timestamp).UTC().String()
	return fmt.Sprintf("Pending msgs: sender: %s, %s at %s", p.Sender, strings.Join(p.GetMsgTypes(), ","), tsStr)
}

func (p ProcessedMsgs) GetMsgStrings() []string {
	msgStrings := make([]string, 0, len(p.Msgs))
	for _, msg := range p.Msgs {
		msgStrings = append(msgStrings, msg.String())
	}
	return msgStrings
}

func (p ProcessedMsgs) GetMsgTypes() []string {
	msgTypes := make([]string, 0, len(p.Msgs))
	for _, msg := range p.Msgs {
		msgTypes = append(msgTypes, sdk.MsgTypeURL(msg))
	}
	return msgTypes
}
