package host

import (
	"context"

	"go.uber.org/zap"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	"github.com/initia-labs/OPinit/x/ophost"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	"github.com/initia-labs/opinit-bots/keys"
	"github.com/initia-labs/opinit-bots/node"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"
)

type BaseHost struct {
	version uint8

	node *node.Node

	bridgeInfo opchildtypes.BridgeInfo

	cfg    nodetypes.NodeConfig
	db     types.DB
	logger *zap.Logger

	ophostQueryClient ophosttypes.QueryClient

	processedMsgs []btypes.ProcessedMsgs
	msgQueue      []sdk.Msg
}

func NewBaseHostV1(cfg nodetypes.NodeConfig,
	db types.DB, logger *zap.Logger, bech32Prefix string,
) *BaseHost {
	appCodec, txConfig, err := GetCodec(bech32Prefix)
	if err != nil {
		panic(err)
	}

	node, err := node.NewNode(cfg, db, logger, appCodec, txConfig)
	if err != nil {
		panic(err)
	}

	h := &BaseHost{
		version: 1,

		node: node,

		cfg:    cfg,
		db:     db,
		logger: logger,

		ophostQueryClient: ophosttypes.NewQueryClient(node.GetRPCClient()),

		processedMsgs: make([]btypes.ProcessedMsgs, 0),
		msgQueue:      make([]sdk.Msg, 0),
	}

	return h
}

func GetCodec(bech32Prefix string) (codec.Codec, client.TxConfig, error) {
	unlock := keys.SetSDKConfigContext(bech32Prefix)
	defer unlock()

	return keys.CreateCodec([]keys.RegisterInterfaces{
		auth.AppModuleBasic{}.RegisterInterfaces,
		ophost.AppModuleBasic{}.RegisterInterfaces,
	})
}

func (b *BaseHost) Initialize(ctx context.Context, startHeight int64, bridgeInfo opchildtypes.BridgeInfo) error {
	err := b.node.Initialize(ctx, startHeight)
	if err != nil {
		return err
	}
	b.SetBridgeInfo(bridgeInfo)
	return nil
}

func (b *BaseHost) Start(ctx context.Context) {
	if b.cfg.ProcessType == nodetypes.PROCESS_TYPE_ONLY_BROADCAST {
		b.logger.Info("host start")
	} else {
		b.logger.Info("host start", zap.Int64("height", b.node.GetHeight()))
	}
	b.node.Start(ctx)
}

func (b BaseHost) BroadcastMsgs(msgs btypes.ProcessedMsgs) {
	if len(msgs.Msgs) == 0 {
		return
	}

	b.node.MustGetBroadcaster().BroadcastMsgs(msgs)
}

func (b BaseHost) ProcessedMsgsToRawKV(msgs []btypes.ProcessedMsgs, delete bool) ([]types.RawKV, error) {
	if len(msgs) == 0 {
		return nil, nil
	}

	return b.node.MustGetBroadcaster().ProcessedMsgsToRawKV(msgs, delete)
}

func (b BaseHost) BridgeId() uint64 {
	return b.bridgeInfo.BridgeId
}

func (b BaseHost) OracleEnabled() bool {
	return b.bridgeInfo.BridgeConfig.OracleEnabled
}

func (b *BaseHost) SetBridgeInfo(bridgeInfo opchildtypes.BridgeInfo) {
	b.bridgeInfo = bridgeInfo
}

func (b BaseHost) BridgeInfo() opchildtypes.BridgeInfo {
	return b.bridgeInfo
}

func (b BaseHost) HasKey() bool {
	return b.node.HasBroadcaster()
}

func (b BaseHost) Height() int64 {
	return b.node.GetHeight()
}

func (b BaseHost) Version() uint8 {
	return b.version
}

func (b BaseHost) Node() *node.Node {
	return b.node
}

func (b BaseHost) Logger() *zap.Logger {
	return b.logger
}

func (b BaseHost) DB() types.DB {
	return b.db
}

/// MsgQueue

func (b BaseHost) GetMsgQueue() []sdk.Msg {
	return b.msgQueue
}

func (b *BaseHost) AppendMsgQueue(msg sdk.Msg) {
	b.msgQueue = append(b.msgQueue, msg)
}

func (b *BaseHost) EmptyMsgQueue() {
	b.msgQueue = b.msgQueue[:0]
}

/// ProcessedMsgs

func (b BaseHost) GetProcessedMsgs() []btypes.ProcessedMsgs {
	return b.processedMsgs
}

func (b *BaseHost) AppendProcessedMsgs(msgs btypes.ProcessedMsgs) {
	b.processedMsgs = append(b.processedMsgs, msgs)
}

func (b *BaseHost) EmptyProcessedMsgs() {
	b.processedMsgs = b.processedMsgs[:0]
}
