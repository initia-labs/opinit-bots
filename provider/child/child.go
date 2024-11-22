package child

import (
	"bytes"

	"go.uber.org/zap"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/authz"

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

	"github.com/pkg/errors"
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
	msgQueue      map[string][]sdk.Msg

	baseAccountIndex     int
	oracleAccountIndex   int
	oracleAccountGranter string
}

func NewBaseChildV1(
	cfg nodetypes.NodeConfig,
	db types.DB,
) *BaseChild {
	appCodec, txConfig, err := GetCodec(cfg.Bech32Prefix)
	if err != nil {
		panic(errors.Wrap(err, "failed to get codec"))
	}

	node, err := node.NewNode(cfg, db, appCodec, txConfig)
	if err != nil {
		panic(errors.Wrap(err, "failed to create node"))
	}

	mk, err := merkle.NewMerkle(ophosttypes.GenerateNodeHash)
	if err != nil {
		panic(errors.Wrap(err, "failed to create merkle"))
	}

	ch := &BaseChild{
		version: 1,

		node: node,
		mk:   mk,

		cfg: cfg,

		opchildQueryClient: opchildtypes.NewQueryClient(node.GetRPCClient()),

		processedMsgs: make([]btypes.ProcessedMsgs, 0),
		msgQueue:      make(map[string][]sdk.Msg),

		baseAccountIndex:   -1,
		oracleAccountIndex: -1,
	}
	return ch
}

func GetCodec(bech32Prefix string) (codec.Codec, client.TxConfig, error) {
	unlock := keys.SetSDKConfigContext(bech32Prefix)
	defer unlock()

	return keys.CreateCodec([]keys.RegisterInterfaces{
		auth.AppModuleBasic{}.RegisterInterfaces,
		authz.RegisterInterfaces,
		opchild.AppModuleBasic{}.RegisterInterfaces,
	})
}

func (b *BaseChild) Initialize(
	ctx types.Context,
	processedHeight int64,
	startOutputIndex uint64,
	bridgeInfo ophosttypes.QueryBridgeResponse,
	keyringConfig *btypes.KeyringConfig,
	oracleKeyringConfig *btypes.KeyringConfig,
	disableDeleteFutureWithdrawals bool,
) (uint64, error) {
	b.SetBridgeInfo(bridgeInfo)

	err := b.node.Initialize(ctx, processedHeight, b.keyringConfigs(keyringConfig, oracleKeyringConfig))
	if err != nil {
		return 0, err
	}

	var l2Sequence uint64
	if b.node.HeightInitialized() {
		if !disableDeleteFutureWithdrawals {
			l2Sequence = 1
			if processedHeight != 0 {
				l2Sequence, err = b.QueryNextL2Sequence(ctx, processedHeight)
				if err != nil {
					return 0, err
				}

				err = merkle.DeleteFutureFinalizedTrees(b.DB(), l2Sequence)
				if err != nil {
					return 0, err
				}
			}
			b.initializeTreeFn = func(blockHeight int64) (bool, error) {
				if processedHeight+1 == blockHeight {
					ctx.Logger().Info("initialize tree", zap.Uint64("index", startOutputIndex))
					err := b.mk.InitializeWorkingTree(startOutputIndex, l2Sequence)
					if err != nil {
						return false, errors.Wrap(err, "failed to initialize working tree")
					}
					return true, nil
				}
				return false, nil
			}
		}

		version := types.MustInt64ToUint64(processedHeight)
		err = merkle.DeleteFutureWorkingTrees(b.DB(), version+1)
		if err != nil {
			return 0, err
		}
	}

	if b.OracleEnabled() && oracleKeyringConfig != nil {
		executors, err := b.QueryExecutors(ctx)
		if err != nil {
			return 0, err
		}

		oracleAddr, err := b.OracleAccountAddressString()
		if err != nil {
			return 0, err
		}

		grants, err := b.QueryGranteeGrants(ctx, oracleAddr)
		if err != nil {
			return 0, err
		}
	GRANTLOOP:
		for _, grant := range grants {
			if grant.Authorization.TypeUrl != "/cosmos.authz.v1beta1.GenericAuthorization" ||
				!bytes.Contains(grant.Authorization.Value, []byte(types.MsgUpdateOracleTypeUrl)) {
				continue
			}

			for _, executor := range executors {
				if grant.Granter == executor && grant.Grantee == oracleAddr {
					b.oracleAccountGranter = grant.Granter
					break GRANTLOOP
				}
			}
		}

		if b.oracleAccountGranter == "" {
			return 0, errors.New("oracle account has no grant")
		}
	}
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

func (b BaseChild) GetMsgQueue() map[string][]sdk.Msg {
	return b.msgQueue
}

func (b *BaseChild) AppendMsgQueue(msg sdk.Msg, sender string) {
	b.msgQueue[sender] = append(b.msgQueue[sender], msg)
}

func (b *BaseChild) EmptyMsgQueue() {
	for sender := range b.msgQueue {
		b.msgQueue[sender] = b.msgQueue[sender][:0]
	}
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

func (b BaseChild) WorkingTree() (merkletypes.TreeInfo, error) {
	if b.mk == nil {
		return merkletypes.TreeInfo{}, errors.New("merkle is not initialized")
	}
	return b.mk.WorkingTree()
}

func (b *BaseChild) keyringConfigs(baseConfig *btypes.KeyringConfig, oracleConfig *btypes.KeyringConfig) []btypes.KeyringConfig {
	var configs []btypes.KeyringConfig
	if baseConfig != nil {
		configs = append(configs, *baseConfig)
		b.baseAccountIndex = len(configs) - 1
	}
	if oracleConfig != nil {
		configs = append(configs, *oracleConfig)
		b.oracleAccountIndex = len(configs) - 1
	}
	return configs
}

func (b BaseChild) BaseAccountAddressString() (string, error) {
	broadcaster, err := b.node.GetBroadcaster()
	if err != nil {
		return "", err
	}
	if b.baseAccountIndex == -1 {
		return "", types.ErrKeyNotSet
	}
	account, err := broadcaster.AccountByIndex(b.baseAccountIndex)
	if err != nil {
		return "", err
	}
	sender := account.GetAddressString()
	return sender, nil
}

func (b BaseChild) OracleAccountAddressString() (string, error) {
	broadcaster, err := b.node.GetBroadcaster()
	if err != nil {
		return "", err
	}
	if b.oracleAccountIndex == -1 {
		return "", types.ErrKeyNotSet
	}
	account, err := broadcaster.AccountByIndex(b.oracleAccountIndex)
	if err != nil {
		return "", err
	}
	sender := account.GetAddressString()
	return sender, nil
}

func (b BaseChild) OracleAccountAddress() (sdk.AccAddress, error) {
	broadcaster, err := b.node.GetBroadcaster()
	if err != nil {
		return nil, err
	}
	if b.oracleAccountIndex == -1 {
		return nil, types.ErrKeyNotSet
	}
	account, err := broadcaster.AccountByIndex(b.oracleAccountIndex)
	if err != nil {
		return nil, err
	}
	sender := account.GetAddress()
	return sender, nil
}
