package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	btypes "github.com/initia-labs/opinit-bots-go/node/broadcaster/types"
	"github.com/initia-labs/opinit-bots-go/types"
)

type DANode interface {
	Start(context.Context)
	HasKey() bool
	CreateBatchMsg([]byte) (sdk.Msg, error)
	BroadcastMsgs(btypes.ProcessedMsgs)
	ProcessedMsgsToRawKV(processedMsgs []btypes.ProcessedMsgs, delete bool) ([]types.RawKV, error)
}

// BatchHeader is the header of a batch
type BatchHeader struct {
	// last l2 block height which is included in the batch
	End uint64 `json:"end"`
	// checksums of all chunks
	Chunks [][]byte `json:"chunks"`
}
