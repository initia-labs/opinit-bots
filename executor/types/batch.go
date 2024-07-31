package types

import (
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
	"github.com/initia-labs/opinit-bots-go/types"
)

type DANode interface {
	GetAddressStr() (string, error)
	HasKey() bool
	BroadcastMsgs(nodetypes.ProcessedMsgs)
	ProcessedMsgsToRawKV(processedMsgs []nodetypes.ProcessedMsgs, delete bool) ([]types.RawKV, error)
}

type BatchHeader struct {
	End    uint64   `json:"end"`
	Chunks [][]byte `json:"chunks"`
}
