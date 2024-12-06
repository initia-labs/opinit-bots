package types

import (
	"time"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	comettypes "github.com/cometbft/cometbft/types"
	"github.com/initia-labs/opinit-bots/types"
)

// EventHandlerArgs is the argument for the event handler
// if the event is from FinalizeBlock, Tx is nil
type EventHandlerArgs struct {
	BlockHeight     int64
	BlockTime       time.Time
	LatestHeight    int64
	TxIndex         int64
	Tx              comettypes.Tx
	EventAttributes []abcitypes.EventAttribute
}

// EventHandlerFn is the event handler function
type EventHandlerFn func(types.Context, EventHandlerArgs) error

// TxHandlerArgs is the argument for the tx handler
type TxHandlerArgs struct {
	BlockHeight  int64
	BlockTime    time.Time
	LatestHeight int64
	TxIndex      int64
	Tx           comettypes.Tx
	Success      bool
}

// TxHandlerFn is the tx handler function
type TxHandlerFn func(types.Context, TxHandlerArgs) error

// BeginBlockArgs is the argument for the begin block handler
type BeginBlockArgs struct {
	BlockID      []byte
	Block        cmtproto.Block
	LatestHeight int64
}

// BeginBlockHandlerFn is the begin block handler function
type BeginBlockHandlerFn func(types.Context, BeginBlockArgs) error

// EndBlockArgs is the argument for the end block handler
type EndBlockArgs struct {
	BlockID      []byte
	Block        cmtproto.Block
	LatestHeight int64
}

// EndBlockHandlerFn is the end block handler function
type EndBlockHandlerFn func(types.Context, EndBlockArgs) error

// RawBlockArgs is the argument for the raw block handler
type RawBlockArgs struct {
	BlockHeight  int64
	LatestHeight int64
	BlockBytes   []byte
}

// RawBlockHandlerFn is the raw block handler function
type RawBlockHandlerFn func(types.Context, RawBlockArgs) error
