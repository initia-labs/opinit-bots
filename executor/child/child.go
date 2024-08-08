package child

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	"github.com/initia-labs/OPinit/x/opchild"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	executortypes "github.com/initia-labs/opinit-bots-go/executor/types"
	"github.com/initia-labs/opinit-bots-go/keys"
	"github.com/initia-labs/opinit-bots-go/merkle"
	"github.com/initia-labs/opinit-bots-go/node"
	btypes "github.com/initia-labs/opinit-bots-go/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
	"github.com/initia-labs/opinit-bots-go/types"
)

type hostNode interface {
	GetAddressStr() (string, error)
	HasKey() bool
	BroadcastMsgs(btypes.ProcessedMsgs)
	ProcessedMsgsToRawKV([]btypes.ProcessedMsgs, bool) ([]types.RawKV, error)
	QueryLastOutput() (*ophosttypes.QueryOutputProposalResponse, error)
	QueryOutput(uint64) (*ophosttypes.QueryOutputProposalResponse, error)

	GetMsgProposeOutput(
		bridgeId uint64,
		outputIndex uint64,
		l2BlockNumber uint64,
		outputRoot []byte,
	) (sdk.Msg, error)
}

type Child struct {
	version uint8

	node *node.Node
	host hostNode
	mk   *merkle.Merkle

	bridgeInfo opchildtypes.BridgeInfo

	nextOutputTime        time.Time
	finalizingBlockHeight uint64
	startTreeIndex        uint64

	cfg    nodetypes.NodeConfig
	db     types.DB
	logger *zap.Logger

	opchildQueryClient opchildtypes.QueryClient

	processedMsgs []btypes.ProcessedMsgs
	msgQueue      []sdk.Msg
}

func NewChild(
	version uint8, cfg nodetypes.NodeConfig,
	db types.DB, logger *zap.Logger, bech32Prefix string,
) *Child {
	appCodec, txConfig, err := GetCodec(bech32Prefix)
	if err != nil {
		panic(err)
	}

	node, err := node.NewNode(cfg, db, logger, appCodec, txConfig)
	if err != nil {
		panic(err)
	}

	mk, err := merkle.NewMerkle(db.WithPrefix([]byte(executortypes.MerkleName)), ophosttypes.GenerateNodeHash)
	if err != nil {
		panic(err)
	}

	ch := &Child{
		version: version,

		node: node,
		mk:   mk,

		cfg:    cfg,
		db:     db,
		logger: logger,

		opchildQueryClient: opchildtypes.NewQueryClient(node.GetRPCClient()),

		processedMsgs: make([]btypes.ProcessedMsgs, 0),
		msgQueue:      make([]sdk.Msg, 0),
	}
	return ch
}

func GetCodec(bech32Prefix string) (codec.Codec, client.TxConfig, error) {
	unlock := keys.SetSDKConfigContext(bech32Prefix)
	defer unlock()

	return keys.CreateCodec([]keys.RegisterInterfaces{
		auth.AppModuleBasic{}.RegisterInterfaces,
		opchild.AppModuleBasic{}.RegisterInterfaces,
	})
}

func (ch *Child) Initialize(startHeight uint64, startOutputIndex uint64, host hostNode, bridgeInfo opchildtypes.BridgeInfo) error {
	err := ch.node.Initialize(startHeight)
	if err != nil {
		return err
	}
	ch.startTreeIndex = startOutputIndex
	ch.host = host
	ch.bridgeInfo = bridgeInfo
	ch.registerHandlers()
	return nil
}

func (ch *Child) Start(ctx context.Context) {
	ch.logger.Info("child start", zap.Uint64("height", ch.node.GetHeight()))
	ch.node.Start(ctx)
}

func (ch *Child) registerHandlers() {
	ch.node.RegisterBeginBlockHandler(ch.beginBlockHandler)
	ch.node.RegisterEventHandler(opchildtypes.EventTypeFinalizeTokenDeposit, ch.finalizeDepositHandler)
	ch.node.RegisterEventHandler(opchildtypes.EventTypeUpdateOracle, ch.updateOracleHandler)
	ch.node.RegisterEventHandler(opchildtypes.EventTypeInitiateTokenWithdrawal, ch.initiateWithdrawalHandler)
	ch.node.RegisterEndBlockHandler(ch.endBlockHandler)
}

func (ch Child) BroadcastMsgs(msgs btypes.ProcessedMsgs) {
	if len(msgs.Msgs) == 0 {
		return
	}

	ch.node.MustGetBroadcaster().BroadcastMsgs(msgs)
}

func (ch Child) ProcessedMsgsToRawKV(msgs []btypes.ProcessedMsgs, delete bool) ([]types.RawKV, error) {
	if len(msgs) == 0 {
		return nil, nil
	}

	return ch.node.MustGetBroadcaster().ProcessedMsgsToRawKV(msgs, delete)
}

func (ch Child) BridgeId() uint64 {
	return ch.bridgeInfo.BridgeId
}

func (ch Child) HasKey() bool {
	return ch.node.HasBroadcaster()
}

func (ch *Child) SetBridgeInfo(bridgeInfo opchildtypes.BridgeInfo) {
	ch.bridgeInfo = bridgeInfo
}

func (ch Child) GetHeight() uint64 {
	return ch.node.GetHeight()
}
