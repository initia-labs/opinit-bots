package batchsubmitter

import (
	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"

	"github.com/cosmos/cosmos-sdk/codec"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ executortypes.DANode = &NoopDA{}

type NoopDA struct {
}

func NewNoopDA() *NoopDA {
	return &NoopDA{}
}

func (n NoopDA) Start(_ types.Context) {}
func (n NoopDA) DB() types.DB          { return nil }
func (n NoopDA) Codec() codec.Codec    { return nil }

func (n NoopDA) HasBroadcaster() bool                               { return false }
func (n NoopDA) CreateBatchMsg(_ []byte) (sdk.Msg, string, error)   { return nil, "", nil }
func (n NoopDA) BroadcastProcessedMsgs(nil ...btypes.ProcessedMsgs) {}
func (n NoopDA) GetNodeStatus() (nodetypes.Status, error)           { return nodetypes.Status{}, nil }
func (n NoopDA) LenProcessedBatchMsgs() (int, error)                { return 0, nil }
func (n NoopDA) LenPendingBatchTxs() (int, error)                   { return 0, nil }
