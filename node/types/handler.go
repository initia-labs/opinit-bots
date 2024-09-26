package types

import (
	"context"
	"time"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	comettypes "github.com/cometbft/cometbft/types"
)

type EventHandlerArgs struct {
	BlockHeight     int64
	BlockTime       time.Time
	LatestHeight    int64
	TxIndex         int64
	EventAttributes []abcitypes.EventAttribute
}

type EventHandlerFn func(context.Context, EventHandlerArgs) error

type TxHandlerArgs struct {
	BlockHeight  int64
	BlockTime    time.Time
	LatestHeight int64
	TxIndex      int64
	Tx           comettypes.Tx
}

type TxHandlerFn func(context.Context, TxHandlerArgs) error

type BeginBlockArgs struct {
	BlockID      []byte
	Block        cmtproto.Block
	LatestHeight int64
}

type BeginBlockHandlerFn func(context.Context, BeginBlockArgs) error

type EndBlockArgs struct {
	BlockID      []byte
	Block        cmtproto.Block
	LatestHeight int64
}

type EndBlockHandlerFn func(context.Context, EndBlockArgs) error

type RawBlockArgs struct {
	BlockHeight  int64
	LatestHeight int64
	BlockBytes   []byte
}

type RawBlockHandlerFn func(context.Context, RawBlockArgs) error
