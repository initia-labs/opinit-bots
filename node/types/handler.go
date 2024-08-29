package types

import (
	"context"
	"time"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	comettypes "github.com/cometbft/cometbft/types"
)

type EventHandlerArgs struct {
	BlockHeight     uint64
	BlockTime       time.Time
	LatestHeight    uint64
	TxIndex         uint64
	EventAttributes []abcitypes.EventAttribute
}

type EventHandlerFn func(context.Context, EventHandlerArgs) error

type TxHandlerArgs struct {
	BlockHeight  uint64
	LatestHeight uint64
	TxIndex      uint64
	Tx           comettypes.Tx
}

type TxHandlerFn func(context.Context, TxHandlerArgs) error

type BeginBlockArgs struct {
	BlockID      []byte
	Block        cmtproto.Block
	LatestHeight uint64
}

type BeginBlockHandlerFn func(context.Context, BeginBlockArgs) error

type EndBlockArgs struct {
	BlockID      []byte
	Block        cmtproto.Block
	LatestHeight uint64
}

type EndBlockHandlerFn func(context.Context, EndBlockArgs) error

type RawBlockArgs struct {
	BlockHeight uint64
	BlockBytes  []byte
}

type RawBlockHandlerFn func(context.Context, RawBlockArgs) error
