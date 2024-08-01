package celestia

import (
	"context"
	"crypto/sha256"
	"fmt"

	"go.uber.org/zap"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	executortypes "github.com/initia-labs/opinit-bots-go/executor/types"
	"github.com/initia-labs/opinit-bots-go/node"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
	"github.com/initia-labs/opinit-bots-go/types"

	"github.com/cometbft/cometbft/crypto/merkle"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"

	inclusion "github.com/celestiaorg/go-square/v2/inclusion"
	sh "github.com/celestiaorg/go-square/v2/share"

	celestiatypes "github.com/initia-labs/opinit-bots-go/types/celestia"
)

type batchNode interface {
	ChainID() string
	UpdateBatchInfo(chain string, submitter string, outputIndex uint64, l2BlockNumber uint64)
}

var _ executortypes.DANode = &Celestia{}

type Celestia struct {
	version uint8

	node  *node.Node
	batch batchNode

	bridgeId  int64
	namespace sh.Namespace

	cfg    nodetypes.NodeConfig
	db     types.DB
	logger *zap.Logger

	processedMsgs []nodetypes.ProcessedMsgs
	msgQueue      []sdk.Msg
}

func NewDACelestia(
	version uint8, cfg nodetypes.NodeConfig,
	db types.DB, logger *zap.Logger, homePath string, batchSubmitter string,
) *Celestia {
	appCodec, txConfig, bech32Prefix := GetCodec()
	node, err := node.NewNode(nodetypes.PROCESS_TYPE_ONLY_BROADCAST, cfg, db, logger, appCodec, txConfig, homePath, bech32Prefix, batchSubmitter)
	if err != nil {
		panic(err)
	}
	node.RegisterBuildTxWithMessages(CelestiaBuildTxWithMessages)

	return &Celestia{
		version: version,

		node: node,

		cfg:    cfg,
		db:     db,
		logger: logger,

		processedMsgs: make([]nodetypes.ProcessedMsgs, 0),
		msgQueue:      make([]sdk.Msg, 0),
	}
}

func GetCodec() (codec.Codec, client.TxConfig, string) {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	amino := codec.NewLegacyAmino()
	std.RegisterLegacyAminoCodec(amino)
	std.RegisterInterfaces(interfaceRegistry)

	protoCodec := codec.NewProtoCodec(interfaceRegistry)
	txConfig := tx.NewTxConfig(protoCodec, tx.DefaultSignModes)

	auth.AppModuleBasic{}.RegisterLegacyAminoCodec(amino)
	auth.AppModuleBasic{}.RegisterInterfaces(interfaceRegistry)
	celestiatypes.RegisterInterfaces(interfaceRegistry)

	return protoCodec, txConfig, "celestia"
}

func (c *Celestia) Initialize(batch batchNode, bridgeId int64) error {
	c.batch = batch
	c.bridgeId = bridgeId
	var err error
	c.namespace, err = sh.NewV0Namespace(c.NamespaceID())
	if err != nil {
		return err
	}
	return nil
}

func (c *Celestia) Start(ctx context.Context, errCh chan error) {
	defer func() {
		if r := recover(); r != nil {
			c.logger.Error("da celestia panic", zap.Any("recover", r))
			errCh <- fmt.Errorf("da celestia panic: %v", r)
		}
	}()

	c.node.Start(ctx, errCh)
}

func (c Celestia) BroadcastMsgs(msgs nodetypes.ProcessedMsgs) {
	if !c.node.HasKey() {
		return
	}

	c.node.BroadcastMsgs(msgs)
}

func (c Celestia) ProcessedMsgsToRawKV(msgs []nodetypes.ProcessedMsgs, delete bool) ([]types.RawKV, error) {
	return c.node.ProcessedMsgsToRawKV(msgs, delete)
}

func (c *Celestia) SetBridgeId(brigeId int64) {
	c.bridgeId = brigeId
}

func (c Celestia) HasKey() bool {
	return c.node.HasKey()
}

func (c Celestia) GetHeight() uint64 {
	return c.node.GetHeight()
}

func (c Celestia) CreateBatchMsg(rawBlob []byte) (sdk.Msg, error) {
	submitter, err := c.node.GetAddressString()
	if err != nil {
		return nil, err
	}
	blob, err := sh.NewV0Blob(c.namespace, rawBlob)
	if err != nil {
		return nil, err
	}
	commitment, err := inclusion.CreateCommitment(blob,
		merkle.HashFromByteSlices,
		// github.com/celestiaorg/celestia-app/pkg/appconsts/v1.SubtreeRootThreshold
		64)
	if err != nil {
		return nil, err
	}
	return &celestiatypes.MsgPayForBlobsWithBlob{
		MsgPayForBlobs: &celestiatypes.MsgPayForBlobs{
			Signer:           submitter,
			Namespaces:       [][]byte{c.namespace.Bytes()},
			ShareCommitments: [][]byte{commitment},
			BlobSizes:        []uint32{uint32(len(blob.Data()))},
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
