package child

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	childprovider "github.com/initia-labs/opinit-bots/provider/child"
)

type hostNode interface {
	QuerySyncedOutput(context.Context, uint64, uint64) (*ophosttypes.QueryOutputProposalResponse, error)
}

type Child struct {
	*childprovider.BaseChild

	host hostNode

	pendingEventsMu *sync.Mutex
	pendingEvents   map[challengertypes.ChallengeId]challengertypes.ChallengeEvent

	eventQueue []challengertypes.ChallengeEvent

	finalizingBlockHeight uint64

	challengeCh chan<- challengertypes.Challenge

	// status info
	lastUpdatedOracleL1Height         uint64
	lastFinalizedDepositL1BlockHeight uint64
	lastFinalizedDepositL1Sequence    uint64
	lastOutputTime                    time.Time
	nextOutputTime                    time.Time
}

func NewChildV1(
	cfg nodetypes.NodeConfig,
	db types.DB, logger *zap.Logger, bech32Prefix string,
	challengeCh chan<- challengertypes.Challenge,
) *Child {
	return &Child{
		BaseChild:       childprovider.NewBaseChildV1(cfg, db, logger, bech32Prefix),
		pendingEventsMu: &sync.Mutex{},
		pendingEvents:   make(map[challengertypes.ChallengeId]challengertypes.ChallengeEvent),
		eventQueue:      make([]challengertypes.ChallengeEvent, 0),

		challengeCh: challengeCh,
	}
}

func (ch *Child) Initialize(startHeight uint64, startOutputIndex uint64, host hostNode, bridgeInfo opchildtypes.BridgeInfo) error {
	err := ch.BaseChild.Initialize(startHeight, startOutputIndex, bridgeInfo)
	if err != nil {
		return err
	}
	ch.host = host
	ch.registerHandlers()

	pendingEvents, err := ch.loadPendingEvents()
	if err != nil {
		return err
	}
	ch.SetPendingEvents(pendingEvents)
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
	kvs := make([]types.RawKV, 0, len(events))
	for _, event := range events {
		var data []byte
		var err error

		if !delete {
			data, err = event.Marshal()
			if err != nil {
				return nil, err
			}
		}
		kvs = append(kvs, types.RawKV{
			Key:   ch.DB().PrefixedKey(challengertypes.PrefixedPendingEvent(event.Id())),
			Value: data,
		})
	}
	return kvs, nil
}
