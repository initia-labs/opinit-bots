package types

import (
	comettypes "github.com/cometbft/cometbft/abci/types"
)

type EventWrapper struct {
	BlockHeight int64
	TxIndex     int64
	EventIndex  int64
	Event       comettypes.Event
}
