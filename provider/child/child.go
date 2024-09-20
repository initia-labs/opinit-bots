package child

import (
	"context"

	"go.uber.org/zap"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	"github.com/initia-labs/OPinit/x/opchild"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	"github.com/initia-labs/opinit-bots/keys"
	"github.com/initia-labs/opinit-bots/merkle"
	"github.com/initia-labs/opinit-bots/node"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"
)

type BaseChild struct {
	version uint8

	node *node.Node
	mk   *merkle.Merkle

	bridgeInfo opchildtypes.BridgeInfo

	initializeTreeFn func(uint64) (bool, error)

	cfg    nodetypes.NodeConfig
	db     types.DB
	logger *zap.Logger

	opchildQueryClient opchildtypes.QueryClient

	processedMsgs []btypes.ProcessedMsgs
	msgQueue      []sdk.Msg
}

func NewBaseChildV1(
	cfg nodetypes.NodeConfig,
	db types.DB, logger *zap.Logger, bech32Prefix string,
) *BaseChild {
	appCodec, txConfig, err := GetCodec(bech32Prefix)
	if err != nil {
		panic(err)
	}

	node, err := node.NewNode(cfg, db, logger, appCodec, txConfig)
	if err != nil {
		panic(err)
	}

	mk, err := merkle.NewMerkle(db.WithPrefix([]byte(types.MerkleName)), ophosttypes.GenerateNodeHash)
	if err != nil {
		panic(err)
	}

	ch := &BaseChild{
		version: 1,

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

func (b *BaseChild) Initialize(ctx context.Context, startHeight uint64, startOutputIndex uint64, bridgeInfo opchildtypes.BridgeInfo) (uint64, error) {
	err := b.node.Initialize(ctx, startHeight)
	if err != nil {
		return 0, err
	}

	var l2Sequence uint64
	if b.node.HeightInitialized() && startOutputIndex != 0 {
		l2Sequence, err = b.QueryNextL2Sequence(ctx, startHeight)
		if err != nil {
			return 0, err
		}

		err = b.mk.DeleteFutureFinalizedTrees(l2Sequence)
		if err != nil {
			return 0, err
		}

		err = b.mk.DeleteFutureWorkingTrees(startHeight + 1)
		if err != nil {
			return 0, err
		}

		b.initializeTreeFn = func(blockHeight uint64) (bool, error) {
			if startHeight+1 == blockHeight {
				b.logger.Info("initialize tree", zap.Uint64("index", startOutputIndex))
				err := b.mk.InitializeWorkingTree(startOutputIndex, 1)
				if err != nil {
					return false, err
				}
				return true, nil
			}
			return false, nil
		}
	}
	b.SetBridgeInfo(bridgeInfo)
	return l2Sequence, nil
}

func (b *BaseChild) Start(ctx context.Context) {
	b.logger.Info("child start", zap.Uint64("height", b.Height()))
	b.node.Start(ctx)
}

func (b BaseChild) BroadcastMsgs(msgs btypes.ProcessedMsgs) {
	if len(msgs.Msgs) == 0 {
		return
	}

	b.node.MustGetBroadcaster().BroadcastMsgs(msgs)
}

func (b BaseChild) ProcessedMsgsToRawKV(msgs []btypes.ProcessedMsgs, delete bool) ([]types.RawKV, error) {
	if len(msgs) == 0 {
		return nil, nil
	}

	return b.node.MustGetBroadcaster().ProcessedMsgsToRawKV(msgs, delete)
}

func (b BaseChild) BridgeId() uint64 {
	return b.bridgeInfo.BridgeId
}

func (b BaseChild) OracleEnabled() bool {
	return b.bridgeInfo.BridgeConfig.OracleEnabled
}

func (b BaseChild) HasKey() bool {
	return b.node.HasBroadcaster()
}

func (b *BaseChild) SetBridgeInfo(bridgeInfo opchildtypes.BridgeInfo) {
	b.bridgeInfo = bridgeInfo
}

func (b BaseChild) BridgeInfo() opchildtypes.BridgeInfo {
	return b.bridgeInfo
}

func (b BaseChild) Height() uint64 {
	return b.node.GetHeight()
}

func (b BaseChild) Version() uint8 {
	return b.version
}

func (b BaseChild) Node() *node.Node {
	return b.node
}

func (b BaseChild) Logger() *zap.Logger {
	return b.logger
}

func (b BaseChild) DB() types.DB {
	return b.db
}

/// MsgQueue

func (b BaseChild) GetMsgQueue() []sdk.Msg {
	return b.msgQueue
}

func (b *BaseChild) AppendMsgQueue(msg sdk.Msg) {
	b.msgQueue = append(b.msgQueue, msg)
}

func (b *BaseChild) EmptyMsgQueue() {
	b.msgQueue = b.msgQueue[:0]
}

/// ProcessedMsgs

func (b BaseChild) GetProcessedMsgs() []btypes.ProcessedMsgs {
	return b.processedMsgs
}

func (b *BaseChild) AppendProcessedMsgs(msgs btypes.ProcessedMsgs) {
	b.processedMsgs = append(b.processedMsgs, msgs)
}

func (b *BaseChild) EmptyProcessedMsgs() {
	b.processedMsgs = b.processedMsgs[:0]
}

/// Merkle

func (b BaseChild) Merkle() *merkle.Merkle {
	return b.mk
}
