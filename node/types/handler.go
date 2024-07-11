package types

import (
	abcitypes "github.com/cometbft/cometbft/abci/types"
	comettypes "github.com/cometbft/cometbft/types"
)

type EventHandlerArgs struct {
	BlockHeight     uint64
	LatestHeight    uint64
	TxIndex         uint64
	EventAttributes []abcitypes.EventAttribute
}

type EventHandlerFn func(EventHandlerArgs) error

type TxHandlerArgs struct {
	BlockHeight  uint64
	LatestHeight uint64
	TxIndex      uint64
	Tx           comettypes.Tx
}

type TxHandlerFn func(TxHandlerArgs) error

type BeginBlockArgs struct {
	BlockHeight  uint64
	LatestHeight uint64
}

type BeginBlockHandlerFn func(BeginBlockArgs) error

type EndBlockArgs struct {
	BlockHeight  uint64
	LatestHeight uint64
}

type EndBlockHandlerFn func(EndBlockArgs) error
