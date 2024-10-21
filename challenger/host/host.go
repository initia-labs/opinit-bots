package host

import (
	"context"
	"time"

	"go.uber.org/zap"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"

	hostprovider "github.com/initia-labs/opinit-bots/provider/host"

	"github.com/initia-labs/opinit-bots/challenger/eventhandler"
	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
)

type challenger interface {
	PendingChallengeToRawKVs([]challengertypes.Challenge, bool) ([]types.RawKV, error)
	SendPendingChallenges([]challengertypes.Challenge)
}

type childNode interface {
	PendingEventsToRawKV([]challengertypes.ChallengeEvent, bool) ([]types.RawKV, error)
	SetPendingEvents([]challengertypes.ChallengeEvent)
}

type Host struct {
	*hostprovider.BaseHost

	child        childNode
	challenger   challenger
	eventHandler *eventhandler.ChallengeEventHandler

	eventQueue              []challengertypes.ChallengeEvent
	outputPendingEventQueue []challengertypes.ChallengeEvent

	// status info
	lastOutputIndex uint64
	lastOutputTime  time.Time
}

func NewHostV1(
	cfg nodetypes.NodeConfig,
	db types.DB, logger *zap.Logger, bech32Prefix string,
) *Host {
	return &Host{
		BaseHost:                hostprovider.NewBaseHostV1(cfg, db, logger, bech32Prefix),
		eventHandler:            eventhandler.NewChallengeEventHandler(db, logger),
		eventQueue:              make([]challengertypes.ChallengeEvent, 0),
		outputPendingEventQueue: make([]challengertypes.ChallengeEvent, 0),
	}
}

func (h *Host) Initialize(ctx context.Context, processedHeight int64, child childNode, bridgeInfo opchildtypes.BridgeInfo, challenger challenger) (time.Time, error) {
	err := h.BaseHost.Initialize(ctx, processedHeight, bridgeInfo)
	if err != nil {
		return time.Time{}, err
	}
	h.child = child
	h.challenger = challenger
	// TODO: ignore l1Sequence less than child's last l1 sequence
	h.registerHandlers()

	err = h.eventHandler.Initialize(bridgeInfo.BridgeConfig.SubmissionInterval)
	if err != nil {
		return time.Time{}, err
	}

	var blockTime time.Time
	if h.Node().HeightInitialized() {
		blockTime, err = h.Node().QueryBlockTime(ctx, h.Node().GetHeight())
		if err != nil {
			return time.Time{}, err
		}
	}
	return blockTime, nil
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
