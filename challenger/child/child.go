package child

import (
	"context"
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
	QueryLastOutput(context.Context, uint64) (*ophosttypes.QueryOutputProposalResponse, error)
	QueryOutput(context.Context, uint64, uint64) (*ophosttypes.QueryOutputProposalResponse, error)
}

type Child struct {
	*childprovider.BaseChild

	host hostNode

	finalizingBlockHeight uint64

	// status info
	lastUpdatedOracleL1Height         uint64
	lastFinalizedDepositL1BlockHeight uint64
	lastFinalizedDepositL1Sequence    uint64
	lastOutputTime                    time.Time
	nextOutputTime                    time.Time

	elemCh    chan<- challengertypes.ChallengeElem
	elemQueue []challengertypes.ChallengeElem
}

func NewChildV1(
	cfg nodetypes.NodeConfig,
	db types.DB, logger *zap.Logger, bech32Prefix string,
	elemCh chan<- challengertypes.ChallengeElem,
) *Child {
	return &Child{
		BaseChild: childprovider.NewBaseChildV1(cfg, db, logger, bech32Prefix),
		elemCh:    elemCh,
		elemQueue: make([]challengertypes.ChallengeElem, 0),
	}
}

func (ch *Child) Initialize(startHeight uint64, startOutputIndex uint64, host hostNode, bridgeInfo opchildtypes.BridgeInfo) error {
	err := ch.BaseChild.Initialize(startHeight, startOutputIndex, bridgeInfo)
	if err != nil {
		return err
	}
	ch.host = host
	ch.registerHandlers()
	return nil
}

func (ch *Child) registerHandlers() {
	ch.Node().RegisterBeginBlockHandler(ch.beginBlockHandler)
	ch.Node().RegisterTxHandler(ch.txHandler)
	ch.Node().RegisterEventHandler(opchildtypes.EventTypeFinalizeTokenDeposit, ch.finalizeDepositHandler)
	ch.Node().RegisterEventHandler(opchildtypes.EventTypeInitiateTokenWithdrawal, ch.initiateWithdrawalHandler)
	ch.Node().RegisterEndBlockHandler(ch.endBlockHandler)
}

func (ch Child) NodeType() challengertypes.NodeType {
	return challengertypes.NodeTypeChild
}
