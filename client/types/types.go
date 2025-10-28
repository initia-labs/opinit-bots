package types

import (
	"cosmossdk.io/api/tendermint/abci"
	"github.com/cometbft/cometbft/libs/bytes"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
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
	CheckTx  ResponseCheckTx   `json:"check_tx"`
	TxResult abci.ExecTxResult `json:"tx_result"`
	Hash     bytes.HexBytes    `json:"hash"`
	Height   int64             `json:"height"`
}

type ResponseCheckTx struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

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
