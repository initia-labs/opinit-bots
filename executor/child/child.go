package child

import (
	"context"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/codec"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"

	childprovider "github.com/initia-labs/opinit-bots/provider/child"

	"github.com/pkg/errors"
)

type hostNode interface {
	DB() types.DB
	Codec() codec.Codec

	HasBroadcaster() bool
	BroadcastProcessedMsgs(...btypes.ProcessedMsgs)

	QueryLastOutput(context.Context, uint64) (*ophosttypes.QueryOutputProposalResponse, error)
	QueryOutput(context.Context, uint64, uint64, int64) (*ophosttypes.QueryOutputProposalResponse, error)

	GetMsgProposeOutput(uint64, uint64, int64, []byte) (sdk.Msg, string, error)
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

	stage           types.CommitDB
	addressIndexMap map[string]uint64
}

func NewChildV1(
	cfg nodetypes.NodeConfig,
	db types.DB,
) *Child {
	return &Child{
		BaseChild:       childprovider.NewBaseChildV1(cfg, db),
		stage:           db.NewStage(),
		addressIndexMap: make(map[string]uint64),
	}
}

func (ch *Child) Initialize(
	ctx types.Context,
	processedHeight int64,
	startOutputIndex uint64,
	host hostNode,
	bridgeInfo ophosttypes.QueryBridgeResponse,
	keyringConfig *btypes.KeyringConfig,
	oracleKeyringConfig *btypes.KeyringConfig,
	disableDeleteFutureWithdrawals bool,
) error {
	l2Sequence, err := ch.BaseChild.Initialize(
		ctx,
		processedHeight,
		startOutputIndex,
		bridgeInfo,
		keyringConfig,
		oracleKeyringConfig,
		disableDeleteFutureWithdrawals,
	)
	if err != nil {
		return errors.Wrap(err, "failed to initialize base child")
	}
	if l2Sequence != 0 {
		err = ch.DeleteFutureWithdrawals(l2Sequence)
		if err != nil {
			return errors.Wrap(err, "failed to delete future withdrawals")
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
