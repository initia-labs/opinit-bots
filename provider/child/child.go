package child

import (
	"errors"

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
	merkletypes "github.com/initia-labs/opinit-bots/merkle/types"
	"github.com/initia-labs/opinit-bots/node"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"
)

type BaseChild struct {
	version uint8

	node *node.Node
	mk   *merkle.Merkle

	bridgeInfo ophosttypes.QueryBridgeResponse

	initializeTreeFn func(int64) (bool, error)

	cfg nodetypes.NodeConfig

	opchildQueryClient opchildtypes.QueryClient

	processedMsgs []btypes.ProcessedMsgs
	msgQueue      []sdk.Msg
	stagingKVs    []types.RawKV
}

func NewBaseChildV1(
	cfg nodetypes.NodeConfig,
	db types.DB,
) *BaseChild {
	appCodec, txConfig, err := GetCodec(cfg.Bech32Prefix)
	if err != nil {
		panic(err)
	}

	node, err := node.NewNode(cfg, db, appCodec, txConfig)
	if err != nil {
		panic(err)
	}

	mk, err := merkle.NewMerkle(ophosttypes.GenerateNodeHash)
	if err != nil {
		panic(err)
	}

	ch := &BaseChild{
		version: 1,

		node: node,
		mk:   mk,

		cfg: cfg,

		opchildQueryClient: opchildtypes.NewQueryClient(node.GetRPCClient()),

		processedMsgs: make([]btypes.ProcessedMsgs, 0),
		msgQueue:      make([]sdk.Msg, 0),
		stagingKVs:    make([]types.RawKV, 0),
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

func (b *BaseChild) Initialize(ctx types.Context, processedHeight int64, startOutputIndex uint64, bridgeInfo ophosttypes.QueryBridgeResponse, keyringConfig *btypes.KeyringConfig) (uint64, error) {
	err := b.node.Initialize(ctx, processedHeight, keyringConfig)
	if err != nil {
		return 0, err
	}

	var l2Sequence uint64
	if b.node.HeightInitialized() {
		l2Sequence, err = b.QueryNextL2Sequence(ctx, processedHeight)
		if err != nil {
			return 0, err
		}

		err = merkle.DeleteFutureFinalizedTrees(b.DB(), l2Sequence)
		if err != nil {
			return 0, err
		}

		version := types.MustInt64ToUint64(processedHeight)
		err = merkle.DeleteFutureWorkingTrees(b.DB(), version+1)
		if err != nil {
			return 0, err
		}

		b.initializeTreeFn = func(blockHeight int64) (bool, error) {
			if processedHeight+1 == blockHeight {
				ctx.Logger().Info("initialize tree", zap.Uint64("index", startOutputIndex))
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

func (b *BaseChild) Start(ctx types.Context) {
	ctx.Logger().Info("child start", zap.Int64("height", b.Height()))
	b.node.Start(ctx)
}
func (b BaseChild) BroadcastProcessedMsgs(batch ...btypes.ProcessedMsgs) {
	if len(batch) == 0 {
		return
	}
	broadcaster := b.node.MustGetBroadcaster()

	for _, processedMsgs := range batch {
		if len(processedMsgs.Msgs) == 0 {
			continue
		}
		broadcaster.BroadcastProcessedMsgs(processedMsgs)
	}
}

func (b BaseChild) DB() types.DB {
	return b.node.DB()
}

func (b BaseChild) Codec() codec.Codec {
	return b.node.Codec()
}

func (b BaseChild) BridgeId() uint64 {
	return b.bridgeInfo.BridgeId
}

func (b BaseChild) OracleEnabled() bool {
	return b.bridgeInfo.BridgeConfig.OracleEnabled
}

func (b BaseChild) HasBroadcaster() bool {
	return b.node.HasBroadcaster()
}

func (b *BaseChild) SetBridgeInfo(bridgeInfo ophosttypes.QueryBridgeResponse) {
	b.bridgeInfo = bridgeInfo
}

func (b BaseChild) BridgeInfo() ophosttypes.QueryBridgeResponse {
	return b.bridgeInfo
}

func (b BaseChild) Height() int64 {
	return b.node.GetHeight()
}

func (b BaseChild) Version() uint8 {
	return b.version
}

func (b BaseChild) Node() *node.Node {
	return b.node
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

func (b *BaseChild) AppendProcessedMsgs(msgs ...btypes.ProcessedMsgs) {
	b.processedMsgs = append(b.processedMsgs, msgs...)
}

func (b *BaseChild) EmptyProcessedMsgs() {
	b.processedMsgs = b.processedMsgs[:0]
}

// / Merkle
func (b BaseChild) Merkle() *merkle.Merkle {
	return b.mk
}

func (b BaseChild) GetWorkingTree() (merkletypes.TreeInfo, error) {
	if b.mk == nil {
		return merkletypes.TreeInfo{}, errors.New("merkle is not initialized")
	}
	return b.mk.GetWorkingTree()
}
