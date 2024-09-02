package host

import (
	"context"

	"go.uber.org/zap"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"

	hostprovider "github.com/initia-labs/opinit-bots/provider/host"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
)

type childNode interface {
	PendingEventsToRawKV([]challengertypes.ChallengeEvent, bool) ([]types.RawKV, error)
	SetPendingEvents([]challengertypes.ChallengeEvent)
}

type Host struct {
	*hostprovider.BaseHost

	child      childNode
	eventQueue []challengertypes.ChallengeEvent
}

func NewHostV1(
	cfg nodetypes.NodeConfig,
	db types.DB, logger *zap.Logger, bech32Prefix string,
) *Host {
	return &Host{
		BaseHost:   hostprovider.NewBaseHostV1(cfg, db, logger, bech32Prefix),
		eventQueue: make([]challengertypes.ChallengeEvent, 0),
	}
}

func (h *Host) Initialize(ctx context.Context, startHeight uint64, child childNode, bridgeInfo opchildtypes.BridgeInfo) error {
	err := h.BaseHost.Initialize(ctx, startHeight, bridgeInfo)
	if err != nil {
		return err
	}
	h.child = child
	// TODO: ignore l1Sequence less than child's last l1 sequence
	h.registerHandlers()
	return nil
}

func (h *Host) registerHandlers() {
	h.Node().RegisterBeginBlockHandler(h.beginBlockHandler)
	h.Node().RegisterTxHandler(h.txHandler)
	h.Node().RegisterEventHandler(ophosttypes.EventTypeInitiateTokenDeposit, h.initiateDepositHandler)
	h.Node().RegisterEventHandler(ophosttypes.EventTypeProposeOutput, h.proposeOutputHandler)
	h.Node().RegisterEndBlockHandler(h.endBlockHandler)
}

func (h *Host) QuerySyncedOutput(ctx context.Context, bridgeId uint64, outputIndex uint64) (*ophosttypes.QueryOutputProposalResponse, error) {
	return h.BaseHost.QueryOutput(ctx, bridgeId, outputIndex, h.Height()-1)
}
