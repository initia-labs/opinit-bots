package types

import (
	"cosmossdk.io/api/tendermint/abci"
	abcitypes "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/bytes"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cometbft/cometbft/types"
)

// Result of block bulk
type ResultBlockBulk struct {
	Blocks [][]byte `json:"blocks"`
}

// Result of raw commit bytes
type ResultRawCommit struct {
	Commit []byte `json:"commit"`
}

// CheckTx and ExecTx results
type ResultBroadcastTxCommit struct {
	CheckTx  ResponseCheckTx `json:"check_tx"`
	TxResult ExecTxResult    `json:"tx_result"`
	Hash     bytes.HexBytes  `json:"hash"`
	Height   int64           `json:"height"`
}

type ResponseCheckTx struct {
	Code      uint32        `protobuf:"varint,1,opt,name=code,proto3" json:"code,omitempty"`
	Data      []byte        `protobuf:"bytes,2,opt,name=data,proto3" json:"data,omitempty"`
	Log       string        `protobuf:"bytes,3,opt,name=log,proto3" json:"log,omitempty"`   // nondeterministic
	Info      string        `protobuf:"bytes,4,opt,name=info,proto3" json:"info,omitempty"` // nondeterministic
	GasWanted int64         `protobuf:"varint,5,opt,name=gas_wanted,proto3" json:"gas_wanted,omitempty"`
	GasUsed   int64         `protobuf:"varint,6,opt,name=gas_used,proto3" json:"gas_used,omitempty"`
	Events    []*abci.Event `protobuf:"bytes,7,rep,name=events,proto3" json:"events,omitempty"`
	Codespace string        `protobuf:"bytes,8,opt,name=codespace,proto3" json:"codespace,omitempty"`

	Address  []byte `protobuf:"bytes,15,opt,name=address,proto3" json:"address,omitempty"`
	Priority int64  `protobuf:"varint,16,opt,name=priority,proto3" json:"priority,omitempty"`
	Sequence uint64 `protobuf:"varint,17,opt,name=sequence,proto3" json:"sequence,omitempty"`
}

type ExecTxResult struct {
	Code      uint32        `protobuf:"varint,1,opt,name=code,proto3" json:"code,omitempty"`
	Data      []byte        `protobuf:"bytes,2,opt,name=data,proto3" json:"data,omitempty"`
	Log       string        `protobuf:"bytes,3,opt,name=log,proto3" json:"log,omitempty"`
	Info      string        `protobuf:"bytes,4,opt,name=info,proto3" json:"info,omitempty"`
	GasWanted int64         `protobuf:"varint,5,opt,name=gas_wanted,proto3" json:"gas_wanted,omitempty"`
	GasUsed   int64         `protobuf:"varint,6,opt,name=gas_used,proto3" json:"gas_used,omitempty"`
	Events    []*abci.Event `protobuf:"bytes,7,rep,name=events,proto3" json:"events,omitempty"`
	Codespace string        `protobuf:"bytes,8,opt,name=codespace,proto3" json:"codespace,omitempty"`
	Signers   []string      `protobuf:"bytes,9,rep,name=signers,proto3" json:"signers,omitempty"`
}

func (r ExecTxResult) ToCoreTypes() abcitypes.ExecTxResult {
	events := make([]abcitypes.Event, len(r.Events))
	for i, event := range r.Events {
		attributes := make([]abcitypes.EventAttribute, len(event.Attributes))
		for j, attribute := range event.Attributes {
			attributes[j] = abcitypes.EventAttribute{
				Key:   attribute.Key,
				Value: attribute.Value,
				Index: attribute.Index,
			}
		}
		events[i] = abcitypes.Event{
			Type:       event.Type_,
			Attributes: attributes,
		}
	}

	return abcitypes.ExecTxResult{
		Code:      r.Code,
		Data:      r.Data,
		Log:       r.Log,
		Info:      r.Info,
		GasWanted: r.GasWanted,
		GasUsed:   r.GasUsed,
		Events:    events,
		Codespace: r.Codespace,
	}
}

type ResultTx struct {
	Hash     bytes.HexBytes `json:"hash"`
	Height   int64          `json:"height"`
	Index    uint32         `json:"index"`
	TxResult ExecTxResult   `json:"tx_result"`
	Tx       types.Tx       `json:"tx"`
	Proof    types.TxProof  `json:"proof,omitempty"`
}

func (r ResultTx) ToCoreTypes() *ctypes.ResultTx {
	return &ctypes.ResultTx{
		Hash:     r.Hash,
		Height:   r.Height,
		Index:    r.Index,
		TxResult: r.TxResult.ToCoreTypes(),
		Tx:       r.Tx,
		Proof:    r.Proof,
	}
}
