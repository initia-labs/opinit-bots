package types

import (
	"time"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	comettypes "github.com/cometbft/cometbft/types"
	"github.com/initia-labs/opinit-bots/types"
)

type EventHandlerArgs struct {
	BlockHeight     int64
	BlockTime       time.Time
	LatestHeight    int64
	TxIndex         int64
	EventAttributes []abcitypes.EventAttribute
}

type EventHandlerFn func(types.Context, EventHandlerArgs) error

type TxHandlerArgs struct {
	BlockHeight  int64
	BlockTime    time.Time
	LatestHeight int64
	TxIndex      int64
	Tx           comettypes.Tx
	Success      bool
}

type TxHandlerFn func(types.Context, TxHandlerArgs) error

type BeginBlockArgs struct {
	BlockID      []byte
	Block        cmtproto.Block
	LatestHeight int64
}

type BeginBlockHandlerFn func(types.Context, BeginBlockArgs) error

type EndBlockArgs struct {
	BlockID      []byte
	Block        cmtproto.Block
	LatestHeight int64
}

type EndBlockHandlerFn func(types.Context, EndBlockArgs) error

type RawBlockArgs struct {
	BlockHeight  int64
	LatestHeight int64
	BlockBytes   []byte
}

type RawBlockHandlerFn func(types.Context, RawBlockArgs) error
