package celestia

import (
	"context"
	"crypto/sha256"
	"errors"

	"go.uber.org/zap"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/x/auth"

	sh "github.com/celestiaorg/go-square/v2/share"

	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	"github.com/initia-labs/opinit-bots/keys"
	"github.com/initia-labs/opinit-bots/node"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"
	celestiatypes "github.com/initia-labs/opinit-bots/types/celestia"
)

type batchNode interface {
	ChainID() string
	UpdateBatchInfo(string, string, uint64, int64)
}

var _ executortypes.DANode = &Celestia{}

type Celestia struct {
	version uint8

	node  *node.Node
	batch batchNode

	bridgeId  uint64
	namespace sh.Namespace

	cfg    nodetypes.NodeConfig
	db     types.DB
	logger *zap.Logger
}

func NewDACelestia(
	version uint8, cfg nodetypes.NodeConfig,
	db types.DB, logger *zap.Logger,
) *Celestia {
	c := &Celestia{
		version: version,

		cfg:    cfg,
		db:     db,
		logger: logger,
	}

	appCodec, txConfig, err := createCodec(cfg.Bech32Prefix)
	if err != nil {
		panic(err)
	}

	node, err := node.NewNode(cfg, db, logger, appCodec, txConfig)
	if err != nil {
		panic(err)
	}

	c.node = node
	return c
}

func createCodec(bech32Prefix string) (codec.Codec, client.TxConfig, error) {
	unlock := keys.SetSDKConfigContext(bech32Prefix)
	defer unlock()

	return keys.CreateCodec([]keys.RegisterInterfaces{
		auth.AppModuleBasic{}.RegisterInterfaces,
		celestiatypes.RegisterInterfaces,
	})
}

func (c *Celestia) Initialize(ctx context.Context, batch batchNode, bridgeId uint64, keyringConfig *btypes.KeyringConfig) error {
	err := c.node.Initialize(ctx, 0, c.keyringConfigs(keyringConfig))
	if err != nil {
		return err
	}

	c.batch = batch
	c.bridgeId = bridgeId
	c.namespace, err = sh.NewV0Namespace(c.NamespaceID())
	if err != nil {
		return err
	}
	return nil
}

func (c *Celestia) RegisterDAHandlers() {
	c.node.RegisterEventHandler("celestia.blob.v1.EventPayForBlobs", c.payForBlobsHandler)
}

func (c *Celestia) Start(ctx context.Context) {
	c.logger.Info("celestia start")
	c.node.Start(ctx)
}

func (c Celestia) BroadcastMsgs(msgs btypes.ProcessedMsgs) {
	if len(msgs.Msgs) == 0 {
		return
	}

	c.node.MustGetBroadcaster().BroadcastMsgs(msgs)
}

func (c Celestia) ProcessedMsgsToRawKV(msgs []btypes.ProcessedMsgs, delete bool) ([]types.RawKV, error) {
	return c.node.MustGetBroadcaster().ProcessedMsgsToRawKV(msgs, delete)
}

func (c *Celestia) SetBridgeId(brigeId uint64) {
	c.bridgeId = brigeId
}

func (c Celestia) HasKey() bool {
	return c.node.HasBroadcaster()
}

func (c Celestia) GetHeight() int64 {
	return c.node.GetHeight()
}

func (c Celestia) NamespaceID() []byte {
	chainIDhash := sha256.Sum256([]byte(c.batch.ChainID()))
	return chainIDhash[:10]
}

func (c Celestia) BaseAccountAddress() (string, error) {
	broadcaster, err := c.node.GetBroadcaster()
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

func (c Celestia) keyringConfigs(baseConfig *btypes.KeyringConfig) []btypes.KeyringConfig {
	var configs []btypes.KeyringConfig
	if baseConfig != nil {
		configs = append(configs, *baseConfig)
	}
	return configs
}
