package celestia

import (
	"crypto/sha256"
	"errors"

	"github.com/cometbft/cometbft/crypto/merkle"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	inclusion "github.com/celestiaorg/go-square/v2/inclusion"
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

	cfg nodetypes.NodeConfig
	db  types.DB

	processedMsgs []btypes.ProcessedMsgs
	msgQueue      []sdk.Msg
}

func NewDACelestia(version uint8, cfg nodetypes.NodeConfig, db types.DB) *Celestia {
	c := &Celestia{
		version: version,

		cfg: cfg,
		db:  db,

		processedMsgs: make([]btypes.ProcessedMsgs, 0),
		msgQueue:      make([]sdk.Msg, 0),
	}

	appCodec, txConfig, err := createCodec(cfg.Bech32Prefix)
	if err != nil {
		panic(err)
	}

	if cfg.BroadcasterConfig != nil {
		cfg.BroadcasterConfig.BuildTxWithMsgs = c.BuildTxWithMessages
		cfg.BroadcasterConfig.MsgsFromTx = c.PendingTxToProcessedMsgs
	}

	node, err := node.NewNode(cfg, db, appCodec, txConfig)
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

func (c *Celestia) Initialize(ctx types.Context, batch batchNode, bridgeId uint64, keyringConfig *btypes.KeyringConfig) error {
	err := c.node.Initialize(ctx, 0, keyringConfig)
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

func (c *Celestia) SetBridgeId(brigeId uint64) {
	c.bridgeId = brigeId
}

func (c Celestia) HasBroadcaster() bool {
	return c.node.HasBroadcaster()
}

func (c Celestia) GetHeight() int64 {
	return c.node.GetHeight()
}

func (c Celestia) CreateBatchMsg(rawBlob []byte) (sdk.Msg, error) {
	submitter, err := c.GetAddressStr()
	if err != nil {
		if errors.Is(err, types.ErrKeyNotSet) {
			return nil, nil
		}
		return nil, err
	}
	blob, err := sh.NewV0Blob(c.namespace, rawBlob)
	if err != nil {
		return nil, err
	}
	commitment, err := inclusion.CreateCommitment(blob,
		merkle.HashFromByteSlices,
		// https://github.com/celestiaorg/celestia-app/blob/4f4d0f7ff1a43b62b232726e52d1793616423df7/pkg/appconsts/v1/app_consts.go#L6
		64,
	)
	if err != nil {
		return nil, err
	}

	dataLength, err := types.SafeIntToUint32(len(blob.Data()))
	if err != nil {
		return nil, err
	}

	return &celestiatypes.MsgPayForBlobsWithBlob{
		MsgPayForBlobs: &celestiatypes.MsgPayForBlobs{
			Signer:           submitter,
			Namespaces:       [][]byte{c.namespace.Bytes()},
			ShareCommitments: [][]byte{commitment},
			BlobSizes:        []uint32{dataLength},
			ShareVersions:    []uint32{uint32(blob.ShareVersion())},
		},
		Blob: &celestiatypes.Blob{
			NamespaceId:      blob.Namespace().ID(),
			Data:             blob.Data(),
			ShareVersion:     uint32(blob.ShareVersion()),
			NamespaceVersion: uint32(blob.Namespace().Version()),
		},
	}, nil
}

func (c Celestia) NamespaceID() []byte {
	chainIDhash := sha256.Sum256([]byte(c.batch.ChainID()))
	return chainIDhash[:10]
}

func (c Celestia) GetAddress() (sdk.AccAddress, error) {
	broadcaster, err := c.node.GetBroadcaster()
	if err != nil {
		return nil, err
	}
	return broadcaster.GetAddress(), nil
}

func (c Celestia) GetAddressStr() (string, error) {
	broadcaster, err := c.node.GetBroadcaster()
	if err != nil {
		return "", err
	}
	return broadcaster.GetAddressString()
}
