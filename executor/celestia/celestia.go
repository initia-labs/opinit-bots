package celestia

import (
	"crypto/sha256"

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

	"github.com/pkg/errors"
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

	cfg nodetypes.NodeConfig
	db  types.DB
}

func NewDACelestia(version uint8, cfg nodetypes.NodeConfig, db types.DB) *Celestia {
	c := &Celestia{
		version: version,

		cfg: cfg,
		db:  db,
	}

	appCodec, txConfig, err := createCodec(cfg.Bech32Prefix)
	if err != nil {
		panic(errors.Wrap(err, "failed to create codec"))
	}

	node, err := node.NewNode(cfg, db, appCodec, txConfig)
	if err != nil {
		panic(errors.Wrap(err, "failed to create node"))
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

func (c *Celestia) Initialize(ctx types.Context, batch batchNode, bridgeId uint64, keyringConfig *btypes.KeyringConfig) error {
	keyringConfig.BuildTxWithMsgs = c.BuildTxWithMsgs
	keyringConfig.MsgsFromTx = c.MsgsFromTx

	err := c.node.Initialize(ctx, 0, c.keyringConfigs(keyringConfig))
	if err != nil {
		return errors.Wrap(err, "failed to initialize node")
	}

	c.batch = batch
	c.bridgeId = bridgeId
	c.namespace, err = sh.NewV0Namespace(c.NamespaceID())
	if err != nil {
		return errors.Wrap(err, "failed to create namespace")
	}
	return nil
}

func (c *Celestia) RegisterDAHandlers() {
	c.node.RegisterEventHandler("celestia.blob.v1.EventPayForBlobs", c.payForBlobsHandler)
}

func (c *Celestia) Start(ctx types.Context) {
	ctx.Logger().Info("celestia start")
	c.node.Start(ctx)
}

func (c Celestia) BroadcastProcessedMsgs(batch ...btypes.ProcessedMsgs) {
	if len(batch) == 0 {
		return
	}
	broadcaster := c.node.MustGetBroadcaster()

	for _, processedMsgs := range batch {
		if len(processedMsgs.Msgs) == 0 {
			continue
		}
		broadcaster.BroadcastProcessedMsgs(processedMsgs)
	}
}

func (c Celestia) DB() types.DB {
	return c.node.DB()
}

func (c Celestia) Codec() codec.Codec {
	return c.node.Codec()
}

func (c *Celestia) SetBridgeId(bridgeId uint64) {
	c.bridgeId = bridgeId
}

func (c Celestia) HasBroadcaster() bool {
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
