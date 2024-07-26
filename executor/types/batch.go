package types

import (
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
	"github.com/initia-labs/opinit-bots-go/types"
)

type DANode interface {
	GetAddressStr() (string, error)
	HasKey() bool
	BroadcastMsgs(nodetypes.ProcessedMsgs)
	RawKVProcessedData([]nodetypes.ProcessedMsgs, bool) ([]types.KV, error)
}

type BatchHeader struct {
	Start  uint64   `json:"start"`
	End    uint64   `json:"end"`
	Chunks [][]byte `json:"chunks"`
}
