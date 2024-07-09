package types

import (
	abcitypes "github.com/cometbft/cometbft/abci/types"
	comettypes "github.com/cometbft/cometbft/types"
)

type EventHandlerArgs struct {
	BlockHeight     int64
	LatestHeight    int64
	TxIndex         int64
	EventAttributes []abcitypes.EventAttribute
}

type EventHandlerFn func(EventHandlerArgs) error

type TxHandlerArgs struct {
	BlockHeight  int64
	LatestHeight int64
	TxIndex      int64
	Tx           comettypes.Tx
}

type TxHandlerFn func(TxHandlerArgs) error

type BeginBlockArgs struct {
	BlockHeight  int64
	LatestHeight int64
}

type BeginBlockHandlerFn func(BeginBlockArgs) error

type EndBlockArgs struct {
	BlockHeight  int64
	LatestHeight int64
}

type EndBlockHandlerFn func(EndBlockArgs) error
