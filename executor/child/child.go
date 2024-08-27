package child

import (
	"context"
	"sync"
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
	QueryOutput(context.Context, uint64, uint64) (*ophosttypes.QueryOutputProposalResponse, error)

	GetMsgProposeOutput(
		bridgeId uint64,
		outputIndex uint64,
		l2BlockNumber uint64,
		outputRoot []byte,
	) (sdk.Msg, error)
}

type Child struct {
	*childprovider.BaseChild

	host hostNode

	nextOutputTime        time.Time
	finalizingBlockHeight uint64

	initializeTreeOnce *sync.Once
	initializeTreeFn   func() error

	// status info
	lastUpdatedOracleL1Height         uint64
	lastFinalizedDepositL1BlockHeight uint64
	lastFinalizedDepositL1Sequence    uint64
	lastOutputTime                    time.Time
}

func NewChildV0(
	cfg nodetypes.NodeConfig,
	db types.DB, logger *zap.Logger, bech32Prefix string,
) *Child {
	return &Child{
		BaseChild:          childprovider.NewBaseChildV0(cfg, db, logger, bech32Prefix),
		initializeTreeOnce: &sync.Once{},
	}
}

func (ch *Child) Initialize(startHeight uint64, startOutputIndex uint64, host hostNode, bridgeInfo opchildtypes.BridgeInfo) error {
	err := ch.Node().Initialize(startHeight)
	if err != nil {
		return err
	}

	if ch.Node().HeightInitialized() && startOutputIndex != 0 {
		ch.initializeTreeFn = func() error {
			ch.Logger().Info("initialize tree", zap.Uint64("index", startOutputIndex))
			err := ch.Merkle().InitializeWorkingTree(startOutputIndex, 1)
			if err != nil {
				return err
			}
			return nil
		}
	}
	ch.host = host
	ch.SetBridgeInfo(bridgeInfo)
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
