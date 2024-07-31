package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
	"github.com/initia-labs/opinit-bots-go/types"
)

type DANode interface {
	HasKey() bool
	CreateBatchMsg([]byte) (sdk.Msg, error)
	BroadcastMsgs(nodetypes.ProcessedMsgs)
	ProcessedMsgsToRawKV([]nodetypes.ProcessedMsgs, bool) ([]types.RawKV, error)
}

type BatchHeader struct {
	End    uint64   `json:"end"`
	Chunks [][]byte `json:"chunks"`
}
