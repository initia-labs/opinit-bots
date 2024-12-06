package child

import (
	"context"
	"time"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/pkg/errors"

	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"

	eventhandler "github.com/initia-labs/opinit-bots/challenger/eventhandler"
	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	childprovider "github.com/initia-labs/opinit-bots/provider/child"
)

type challenger interface {
	DB() types.DB
	SendPendingChallenges([]challengertypes.Challenge)
}

type hostNode interface {
	QuerySyncedOutput(context.Context, uint64, uint64) (*ophosttypes.QueryOutputProposalResponse, error)
}

type Child struct {
	*childprovider.BaseChild

	host         hostNode
	challenger   challenger
	eventHandler *eventhandler.ChallengeEventHandler

	eventQueue []challengertypes.ChallengeEvent

	finalizingBlockHeight int64

	// status info
	lastUpdatedOracleL1Height         int64
	lastFinalizedDepositL1BlockHeight int64
	lastFinalizedDepositL1Sequence    uint64
	lastOutputTime                    time.Time
	nextOutputTime                    time.Time

	stage types.CommitDB
}

func NewChildV1(
	cfg nodetypes.NodeConfig,
	db types.DB,
) *Child {
	return &Child{
		BaseChild:    childprovider.NewBaseChildV1(cfg, db),
		eventHandler: eventhandler.NewChallengeEventHandler(db),
		eventQueue:   make([]challengertypes.ChallengeEvent, 0),
		stage:        db.NewStage(),
	}
}

func (ch *Child) Initialize(
	ctx types.Context,
	processedHeight int64,
	startOutputIndex uint64,
	host hostNode,
	bridgeInfo ophosttypes.QueryBridgeResponse,
	challenger challenger,
) (time.Time, error) {
	_, err := ch.BaseChild.Initialize(
		ctx,
		processedHeight,
		startOutputIndex,
		bridgeInfo,
		nil,
		nil,
		true,
	)
	if err != nil {
		return time.Time{}, errors.Wrap(err, "failed to initialize base child")
	}
	ch.host = host
	ch.challenger = challenger
	ch.registerHandlers()

	err = ch.eventHandler.Initialize(bridgeInfo.BridgeConfig.SubmissionInterval)
	if err != nil {
		return time.Time{}, errors.Wrap(err, "failed to initialize event handler")
	}

	var blockTime time.Time

	// only called when `ResetHeight` was executed.
	if ch.Node().HeightInitialized() {
		blockTime, err = ch.Node().QueryBlockTime(ctx, ch.Node().GetHeight())
		if err != nil {
			return time.Time{}, errors.Wrap(err, "failed to query block time")
		}
	}
	return blockTime, nil
}

func (ch *Child) registerHandlers() {
	ch.Node().RegisterBeginBlockHandler(ch.beginBlockHandler)
	ch.Node().RegisterTxHandler(ch.txHandler)
	ch.Node().RegisterEventHandler(opchildtypes.EventTypeFinalizeTokenDeposit, ch.finalizeDepositHandler)
	ch.Node().RegisterEventHandler(opchildtypes.EventTypeInitiateTokenWithdrawal, ch.initiateWithdrawalHandler)
	ch.Node().RegisterEndBlockHandler(ch.endBlockHandler)
}

func (ch *Child) SetPendingEvents(events []challengertypes.ChallengeEvent) {
	ch.eventHandler.SetPendingEvents(events)
}
