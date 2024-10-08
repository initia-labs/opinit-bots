package child

import (
	"context"
	"time"

	"go.uber.org/zap"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"

	eventhandler "github.com/initia-labs/opinit-bots/challenger/eventhandler"
	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	childprovider "github.com/initia-labs/opinit-bots/provider/child"
)

type challenger interface {
	PendingChallengeToRawKVs([]challengertypes.Challenge, bool) ([]types.RawKV, error)
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
}

func NewChildV1(
	cfg nodetypes.NodeConfig,
	db types.DB, logger *zap.Logger, bech32Prefix string,
) *Child {
	return &Child{
		BaseChild:    childprovider.NewBaseChildV1(cfg, db, logger, bech32Prefix),
		eventHandler: eventhandler.NewChallengeEventHandler(db, logger),
		eventQueue:   make([]challengertypes.ChallengeEvent, 0),
	}
}

func (ch *Child) Initialize(ctx context.Context, processedHeight int64, startOutputIndex uint64, host hostNode, bridgeInfo opchildtypes.BridgeInfo, challenger challenger) error {
	_, err := ch.BaseChild.Initialize(ctx, processedHeight, startOutputIndex, bridgeInfo)
	if err != nil {
		return err
	}
	ch.host = host
	ch.challenger = challenger
	ch.registerHandlers()

	err = ch.eventHandler.Initialize(bridgeInfo.BridgeConfig.SubmissionInterval)
	if err != nil {
		return err
	}
	return nil
}

func (ch *Child) registerHandlers() {
	ch.Node().RegisterBeginBlockHandler(ch.beginBlockHandler)
	ch.Node().RegisterTxHandler(ch.txHandler)
	ch.Node().RegisterEventHandler(opchildtypes.EventTypeFinalizeTokenDeposit, ch.finalizeDepositHandler)
	ch.Node().RegisterEventHandler(opchildtypes.EventTypeInitiateTokenWithdrawal, ch.initiateWithdrawalHandler)
	ch.Node().RegisterEndBlockHandler(ch.endBlockHandler)
}

func (ch *Child) PendingEventsToRawKV(events []challengertypes.ChallengeEvent, delete bool) ([]types.RawKV, error) {
	return ch.eventHandler.PendingEventsToRawKV(events, delete)
}

func (ch *Child) SetPendingEvents(events []challengertypes.ChallengeEvent) {
	ch.eventHandler.SetPendingEvents(events)
}
