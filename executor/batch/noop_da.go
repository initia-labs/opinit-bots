package batch

import (
	"context"

	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ executortypes.DANode = &NoopDA{}

type NoopDA struct {
}

func NewNoopDA() *NoopDA {
	return &NoopDA{}
}

func (n NoopDA) Start(_ context.Context)                  {}
func (n NoopDA) HasKey() bool                             { return false }
func (n NoopDA) CreateBatchMsg(_ []byte) (sdk.Msg, error) { return nil, nil }
func (n NoopDA) BroadcastMsgs(nil btypes.ProcessedMsgs)   {}
func (n NoopDA) ProcessedMsgsToRawKV(_ []btypes.ProcessedMsgs, _ bool) ([]types.RawKV, error) {
	return nil, nil
}
func (n NoopDA) GetNodeStatus() nodetypes.Status { return nodetypes.Status{} }
