package child

import (
	"bytes"
	"context"
	"errors"

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

	cfg    nodetypes.NodeConfig
	db     types.DB
	logger *zap.Logger

	opchildQueryClient opchildtypes.QueryClient

	processedMsgs []btypes.ProcessedMsgs
	msgQueue      map[string][]sdk.Msg

	baseAccountIndex     int
	oracleAccountIndex   int
	oracleAccountGranter string
}

func NewBaseChildV1(
	cfg nodetypes.NodeConfig,
	db types.DB, logger *zap.Logger,
) *BaseChild {
	appCodec, txConfig, err := GetCodec(cfg.Bech32Prefix)
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
	ctx context.Context,
	processedHeight int64,
	startOutputIndex uint64,
	bridgeInfo ophosttypes.QueryBridgeResponse,
	keyringConfig *btypes.KeyringConfig,
	oracleKeyringConfig *btypes.KeyringConfig,
) (uint64, error) {
	b.SetBridgeInfo(bridgeInfo)

	err := b.node.Initialize(ctx, processedHeight, b.keyringConfigs(keyringConfig, oracleKeyringConfig))
	if err != nil {
		return 0, err
	}

	var l2Sequence uint64
	if b.node.HeightInitialized() {
		l2Sequence, err = b.QueryNextL2Sequence(ctx, processedHeight)
		if err != nil {
			return 0, err
		}

		err = b.mk.DeleteFutureFinalizedTrees(l2Sequence)
		if err != nil {
			return 0, err
		}

		version := types.MustInt64ToUint64(processedHeight)
		err = b.mk.DeleteFutureWorkingTrees(version + 1)
		if err != nil {
			return 0, err
		}

		b.initializeTreeFn = func(blockHeight int64) (bool, error) {
			if processedHeight+1 == blockHeight {
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
				!bytes.Contains(grant.Authorization.Value, []byte("/opinit.opchild.v1.MsgUpdateOracle")) {
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

func (b *BaseChild) Start(ctx context.Context) {
	b.logger.Info("child start", zap.Int64("height", b.Height()))
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

func (b BaseChild) Logger() *zap.Logger {
	return b.logger
}

func (b BaseChild) DB() types.DB {
	return b.db
}

/// MsgQueue

func (b BaseChild) GetMsgQueue() map[string][]sdk.Msg {
	return b.msgQueue
}

func (b *BaseChild) AppendMsgQueue(msg sdk.Msg, sender string) {
	if b.msgQueue[sender] == nil {
		b.msgQueue[sender] = make([]sdk.Msg, 0)
	}
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

func (b BaseChild) GetWorkingTreeIndex() (uint64, error) {
	if b.mk == nil {
		return 0, errors.New("merkle is not initialized")
	}
	return b.mk.GetWorkingTreeIndex()
}

func (b BaseChild) GetStartLeafIndex() (uint64, error) {
	if b.mk == nil {
		return 0, errors.New("merkle is not initialized")
	}
	return b.mk.GetStartLeafIndex()
}

func (b BaseChild) GetWorkingTreeLeafCount() (uint64, error) {
	if b.mk == nil {
		return 0, errors.New("merkle is not initialized")
	}
	return b.mk.GetWorkingTreeLeafCount()
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
		return "", nil
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
		return "", nil
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
		return nil, nil
	}
	account, err := broadcaster.AccountByIndex(b.oracleAccountIndex)
	if err != nil {
		return nil, err
	}
	sender := account.GetAddress()
	return sender, nil
}
