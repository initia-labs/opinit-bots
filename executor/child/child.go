package child

import (
	"context"
	"fmt"
	"time"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	executortypes "github.com/initia-labs/opinit-bots-go/executor/types"
	"github.com/initia-labs/opinit-bots-go/merkle"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
	"github.com/initia-labs/opinit-bots-go/types"
	"go.uber.org/zap"

	"github.com/initia-labs/opinit-bots-go/node"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	initiaapp "github.com/initia-labs/initia/app"

	"github.com/initia-labs/OPinit/x/opchild"
	"github.com/initia-labs/initia/app/params"
)

type hostNode interface {
	GetAddressStr() (string, error)
	HasKey() bool
	BroadcastMsgs(nodetypes.ProcessedMsgs)
	ProcessedMsgsToRawKV([]nodetypes.ProcessedMsgs, bool) ([]types.RawKV, error)
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

	cfg    nodetypes.NodeConfig
	db     types.DB
	logger *zap.Logger

	opchildQueryClient opchildtypes.QueryClient

	processedMsgs []nodetypes.ProcessedMsgs
	msgQueue      []sdk.Msg
}

func NewChild(
	version uint8, cfg nodetypes.NodeConfig,
	db types.DB, logger *zap.Logger,
) *Child {
	appCodec, txConfig, bech32Prefix, err := getCodec(cfg.ChainID)
	if err != nil {
		panic(err)
	}
	node, err := node.NewNode(cfg, db, logger, appCodec, txConfig, bech32Prefix)
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

		opchildQueryClient: opchildtypes.NewQueryClient(node),

		processedMsgs: make([]nodetypes.ProcessedMsgs, 0),
		msgQueue:      make([]sdk.Msg, 0),
	}
	return ch
}

func getCodec(chainID string) (codec.Codec, client.TxConfig, string, error) {
	switch chainID {
	case "minimove-1", "miniwasm-1":
		encodingConfig := params.MakeEncodingConfig()
		appCodec := encodingConfig.Codec
		txConfig := encodingConfig.TxConfig

		std.RegisterInterfaces(encodingConfig.InterfaceRegistry)
		auth.AppModuleBasic{}.RegisterInterfaces(encodingConfig.InterfaceRegistry)
		opchild.AppModuleBasic{}.RegisterInterfaces(encodingConfig.InterfaceRegistry)
		return appCodec, txConfig, initiaapp.AccountAddressPrefix, nil
	}

	return nil, nil, "", fmt.Errorf("unsupported chain id: %s", chainID)
}

func (ch *Child) Initialize(host hostNode, bridgeInfo opchildtypes.BridgeInfo) error {
	ch.host = host
	ch.bridgeInfo = bridgeInfo

	ch.registerHandlers()
	return nil
}

func (ch *Child) Start(ctx context.Context, errCh chan error) {
	defer func() {
		if r := recover(); r != nil {
			ch.logger.Error("child panic", zap.Any("recover", r))
			errCh <- fmt.Errorf("child panic: %v", r)
		}
	}()

	ch.node.Start(ctx, errCh, nodetypes.PROCESS_TYPE_DEFAULT)
}

func (ch *Child) registerHandlers() {
	ch.node.RegisterBeginBlockHandler(ch.beginBlockHandler)
	ch.node.RegisterEventHandler(opchildtypes.EventTypeFinalizeTokenDeposit, ch.finalizeDepositHandler)
	ch.node.RegisterEventHandler(opchildtypes.EventTypeUpdateOracle, ch.updateOracleHandler)
	ch.node.RegisterEventHandler(opchildtypes.EventTypeInitiateTokenWithdrawal, ch.initiateWithdrawalHandler)
	ch.node.RegisterEndBlockHandler(ch.endBlockHandler)
}

func (ch Child) BroadcastMsgs(msgs nodetypes.ProcessedMsgs) {
	if !ch.node.HasKey() {
		return
	}

	ch.node.BroadcastMsgs(msgs)
}

func (ch Child) ProcessedMsgsToRawKV(msgs []nodetypes.ProcessedMsgs, delete bool) ([]types.RawKV, error) {
	return ch.node.ProcessedMsgsToRawKV(msgs, delete)
}

func (ch Child) BridgeId() uint64 {
	return ch.bridgeInfo.BridgeId
}

func (ch Child) HasKey() bool {
	return ch.node.HasKey()
}

func (ch *Child) SetBridgeInfo(bridgeInfo opchildtypes.BridgeInfo) {
	ch.bridgeInfo = bridgeInfo
}

func (ch Child) GetHeight() uint64 {
	return ch.node.GetHeight()
}
