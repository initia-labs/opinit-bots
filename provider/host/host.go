package host

import (
	"go.uber.org/zap"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	"github.com/initia-labs/OPinit/x/ophost"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	"github.com/initia-labs/opinit-bots/keys"
	"github.com/initia-labs/opinit-bots/node"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"

	"github.com/pkg/errors"
)

type BaseHost struct {
	version uint8

	node *node.Node

	bridgeInfo ophosttypes.QueryBridgeResponse

	cfg nodetypes.NodeConfig

	ophostQueryClient ophosttypes.QueryClient

	processedMsgs []btypes.ProcessedMsgs
	msgQueue      map[string][]sdk.Msg
}

func NewBaseHostV1(cfg nodetypes.NodeConfig, db types.DB) *BaseHost {
	appCodec, txConfig, err := GetCodec(cfg.Bech32Prefix)
	if err != nil {
		panic(errors.Wrap(err, "failed to get codec"))
	}

	node, err := node.NewNode(cfg, db, appCodec, txConfig)
	if err != nil {
		panic(errors.Wrap(err, "failed to create node"))
	}

	h := &BaseHost{
		version: 1,

		node: node,

		cfg: cfg,

		ophostQueryClient: ophosttypes.NewQueryClient(node.GetRPCClient()),

		processedMsgs: make([]btypes.ProcessedMsgs, 0),
		msgQueue:      make(map[string][]sdk.Msg),
	}

	return h
}

func GetCodec(bech32Prefix string) (codec.Codec, client.TxConfig, error) {
	unlock := keys.SetSDKConfigContext(bech32Prefix)
	defer unlock()

	_, codec, txConfig, err := keys.CreateCodec([]keys.RegisterInterfaces{
		auth.AppModuleBasic{}.RegisterInterfaces,
		ophost.AppModuleBasic{}.RegisterInterfaces,
	})
	return codec, txConfig, err
}

func (b *BaseHost) Initialize(ctx types.Context, processedHeight int64, bridgeInfo ophosttypes.QueryBridgeResponse, keyringConfig *btypes.KeyringConfig) error {
	err := b.node.Initialize(ctx, processedHeight, b.keyringConfigs(keyringConfig))
	if err != nil {
		return errors.Wrap(err, "failed to initialize node")
	}
	b.SetBridgeInfo(bridgeInfo)
	return nil
}

func (b *BaseHost) Start(ctx types.Context) {
	if b.cfg.ProcessType == nodetypes.PROCESS_TYPE_ONLY_BROADCAST {
		ctx.Logger().Info("host start")
	} else {
		ctx.Logger().Info("host start", zap.Int64("height", b.node.GetHeight()))
	}
	b.node.Start(ctx)
}

func (b BaseHost) BroadcastProcessedMsgs(batch ...btypes.ProcessedMsgs) {
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

func (b BaseHost) DB() types.DB {
	return b.node.DB()
}

func (b BaseHost) Codec() codec.Codec {
	return b.node.Codec()
}

func (b BaseHost) BridgeId() uint64 {
	return b.bridgeInfo.BridgeId
}

func (b BaseHost) ChainId() string {
	return b.cfg.ChainID
}

func (b BaseHost) OracleEnabled() bool {
	return b.bridgeInfo.BridgeConfig.OracleEnabled
}

func (b *BaseHost) UpdateOracleEnabled(oracleEnabled bool) {
	b.bridgeInfo.BridgeConfig.OracleEnabled = oracleEnabled
}

func (b *BaseHost) UpdateBatchInfo(batchInfo ophosttypes.BatchInfo) {
	b.bridgeInfo.BridgeConfig.BatchInfo = batchInfo
}

func (b *BaseHost) UpdateProposer(proposer string) {
	b.bridgeInfo.BridgeConfig.Proposer = proposer
}

func (b *BaseHost) UpdateChallenger(challenger string) {
	b.bridgeInfo.BridgeConfig.Challenger = challenger
}

func (b *BaseHost) SetBridgeInfo(bridgeInfo ophosttypes.QueryBridgeResponse) {
	b.bridgeInfo = bridgeInfo
}

func (b BaseHost) BridgeInfo() ophosttypes.QueryBridgeResponse {
	return b.bridgeInfo
}

func (b BaseHost) HasBroadcaster() bool {
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

/// MsgQueue

func (b BaseHost) GetMsgQueue() map[string][]sdk.Msg {
	return b.msgQueue
}

func (b *BaseHost) AppendMsgQueue(msg sdk.Msg, sender string) {
	b.msgQueue[sender] = append(b.msgQueue[sender], msg)
}

func (b *BaseHost) EmptyMsgQueue() {
	for sender := range b.msgQueue {
		b.msgQueue[sender] = b.msgQueue[sender][:0]
	}
}

/// ProcessedMsgs

func (b BaseHost) GetProcessedMsgs() []btypes.ProcessedMsgs {
	return b.processedMsgs
}

func (b *BaseHost) AppendProcessedMsgs(msgs ...btypes.ProcessedMsgs) {
	b.processedMsgs = append(b.processedMsgs, msgs...)
}

func (b *BaseHost) EmptyProcessedMsgs() {
	b.processedMsgs = b.processedMsgs[:0]
}

func (b BaseHost) BaseAccountAddressString() (string, error) {
	broadcaster, err := b.node.GetBroadcaster()
	if err != nil {
		if errors.Is(err, types.ErrKeyNotSet) {
			return "", nil
		}
		return "", err
	}
	account, err := broadcaster.AccountByIndex(0)
	if err != nil {
		return "", err
	}
	sender := account.GetAddressString()
	return sender, nil
}

func (b BaseHost) keyringConfigs(baseConfig *btypes.KeyringConfig) []btypes.KeyringConfig {
	var configs []btypes.KeyringConfig
	if baseConfig != nil {
		configs = append(configs, *baseConfig)
	}
	return configs
}
