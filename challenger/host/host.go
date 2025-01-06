package host

import (
	"context"
	"time"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/pkg/errors"

	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"

	hostprovider "github.com/initia-labs/opinit-bots/provider/host"

	"github.com/initia-labs/opinit-bots/challenger/eventhandler"
	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
)

type challenger interface {
	DB() types.DB
	SendPendingChallenges([]challengertypes.Challenge)
}

type childNode interface {
	DB() types.DB
	SetPendingEvents([]challengertypes.ChallengeEvent)
	Height() int64
	QueryNextL1Sequence(context.Context, int64) (uint64, error)
}

type Host struct {
	*hostprovider.BaseHost

	child        childNode
	challenger   challenger
	eventHandler *eventhandler.ChallengeEventHandler

	initialL1Sequence uint64

	eventQueue              []challengertypes.ChallengeEvent
	outputPendingEventQueue []challengertypes.ChallengeEvent

	// status info
	lastOutputIndex uint64
	lastOutputTime  time.Time

	stage types.CommitDB
}

func NewHostV1(
	cfg nodetypes.NodeConfig,
	db types.DB,
) *Host {
	return &Host{
		BaseHost:                hostprovider.NewBaseHostV1(cfg, db),
		eventHandler:            eventhandler.NewChallengeEventHandler(db),
		eventQueue:              make([]challengertypes.ChallengeEvent, 0),
		outputPendingEventQueue: make([]challengertypes.ChallengeEvent, 0),
		stage:                   db.NewStage(),
	}
}

func (h *Host) Initialize(ctx types.Context, processedHeight int64, child childNode, bridgeInfo ophosttypes.QueryBridgeResponse, challenger challenger) (time.Time, error) {
	err := h.BaseHost.Initialize(ctx, processedHeight, bridgeInfo, nil)
	if err != nil {
		return time.Time{}, errors.Wrap(err, "failed to initialize base host")
	}
	h.child = child
	h.challenger = challenger

	h.initialL1Sequence, err = h.child.QueryNextL1Sequence(ctx, h.child.Height())
	if err != nil {
		return time.Time{}, errors.Wrap(err, "failed to query next l1 sequence")
	}
	h.registerHandlers()

	err = h.eventHandler.Initialize(bridgeInfo.BridgeConfig.SubmissionInterval)
	if err != nil {
		return time.Time{}, errors.Wrap(err, "failed to initialize event handler")
	}

	var blockTime time.Time

	// only called when `ResetHeight` was executed.
	if h.Node().HeightInitialized() {
		blockTime, err = h.Node().QueryBlockTime(ctx, h.Node().GetHeight())
		if err != nil {
			return time.Time{}, errors.Wrap(err, "failed to query block time")
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
