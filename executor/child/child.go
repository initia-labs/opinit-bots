package child

import (
	"context"
	"time"

	"go.uber.org/zap"

	sdk "github.com/cosmos/cosmos-sdk/types"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"

	childprovider "github.com/initia-labs/opinit-bots/provider/child"
)

type hostNode interface {
	GetAddressStr() (string, error)
	HasKey() bool
	BroadcastMsgs(btypes.ProcessedMsgs)
	ProcessedMsgsToRawKV([]btypes.ProcessedMsgs, bool) ([]types.RawKV, error)
	QueryLastOutput(context.Context, uint64) (*ophosttypes.QueryOutputProposalResponse, error)
	QueryOutput(context.Context, uint64, uint64, int64) (*ophosttypes.QueryOutputProposalResponse, error)

	GetMsgProposeOutput(uint64, uint64, int64, []byte) (sdk.Msg, error)
}

type Child struct {
	*childprovider.BaseChild

	host hostNode

	nextOutputTime        time.Time
	finalizingBlockHeight int64

	// status info
	lastUpdatedOracleL1Height         int64
	lastFinalizedDepositL1BlockHeight int64
	lastFinalizedDepositL1Sequence    uint64
	lastOutputTime                    time.Time
}

func NewChildV1(
	cfg nodetypes.NodeConfig,
	db types.DB, logger *zap.Logger, bech32Prefix string,
) *Child {
	return &Child{
		BaseChild: childprovider.NewBaseChildV1(cfg, db, logger, bech32Prefix),
	}
}

func (ch *Child) Initialize(ctx context.Context, startHeight int64, startOutputIndex uint64, host hostNode, bridgeInfo opchildtypes.BridgeInfo) error {
	l2Sequence, err := ch.BaseChild.Initialize(ctx, startHeight, startOutputIndex, bridgeInfo)
	if err != nil {
		return err
	}
	if l2Sequence != 0 {
		err = ch.DeleteFutureWithdrawals(l2Sequence)
		if err != nil {
			return err
		}
	}

	ch.host = host
	ch.registerHandlers()
	return nil
}

func (ch *Child) registerHandlers() {
	ch.Node().RegisterBeginBlockHandler(ch.beginBlockHandler)
	ch.Node().RegisterEventHandler(opchildtypes.EventTypeFinalizeTokenDeposit, ch.finalizeDepositHandler)
	ch.Node().RegisterEventHandler(opchildtypes.EventTypeUpdateOracle, ch.updateOracleHandler)
	ch.Node().RegisterEventHandler(opchildtypes.EventTypeInitiateTokenWithdrawal, ch.initiateWithdrawalHandler)
	ch.Node().RegisterEndBlockHandler(ch.endBlockHandler)
}
