package child

import (
	"context"
	"errors"
	"io"
	"os"
	"time"

	"cosmossdk.io/core/address"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	executortypes "github.com/initia-labs/opinit-bots-go/executor/types"
	"github.com/initia-labs/opinit-bots-go/merkle"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
	"github.com/initia-labs/opinit-bots-go/types"
	"go.uber.org/zap"

	"github.com/initia-labs/opinit-bots-go/node"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type hostNode interface {
	GetAddressStr() (string, error)
	AccountCodec() address.Codec
	HasKey() bool
	BroadcastMsgs(nodetypes.ProcessedMsgs)
	RawKVProcessedData([]nodetypes.ProcessedMsgs, bool) ([]types.KV, error)
	QueryLastOutput() (*ophosttypes.QueryOutputProposalResponse, error)
	QueryOutput(uint64) (*ophosttypes.QueryOutputProposalResponse, error)
	QueryBatchInfos() (*ophosttypes.QueryBatchInfosResponse, error)
}

type compressionFunc interface {
	Write([]byte) (int, error)
	Reset(io.Writer)
	Close() error
}

type Child struct {
	version uint8

	node *node.Node
	host hostNode
	da   executortypes.DANode
	mk   *merkle.Merkle

	bridgeInfo opchildtypes.BridgeInfo

	nextOutputTime        time.Time
	finalizingBlockHeight uint64

	cfg      nodetypes.NodeConfig
	batchCfg executortypes.BatchConfig
	db       types.DB
	logger   *zap.Logger

	cdc codec.Codec
	ac  address.Codec

	opchildQueryClient opchildtypes.QueryClient

	batchInfos  []ophosttypes.BatchInfoWithOutput
	batchWriter compressionFunc
	batchFile   *os.File
	batchHeader *executortypes.BatchHeader

	processedMsgs      []nodetypes.ProcessedMsgs
	msgQueue           []sdk.Msg
	batchProcessedMsgs []nodetypes.ProcessedMsgs

	homePath string
}

func NewChild(version uint8, cfg nodetypes.NodeConfig, batchCfg executortypes.BatchConfig, db types.DB, logger *zap.Logger, cdc codec.Codec, txConfig client.TxConfig, homePath string) *Child {
	node, err := node.NewNode(cfg, db, logger, cdc, txConfig)
	if err != nil {
		panic(err)
	}

	mk := merkle.NewMerkle(db.WithPrefix([]byte(executortypes.MerkleName)), ophosttypes.GenerateNodeHash)

	ch := &Child{
		version: version,

		node: node,
		mk:   mk,

		cfg:      cfg,
		batchCfg: batchCfg,
		db:       db,
		logger:   logger,

		cdc: cdc,
		ac:  cdc.InterfaceRegistry().SigningContext().AddressCodec(),

		opchildQueryClient: opchildtypes.NewQueryClient(node),

		processedMsgs: make([]nodetypes.ProcessedMsgs, 0),
		msgQueue:      make([]sdk.Msg, 0),

		batchProcessedMsgs: make([]nodetypes.ProcessedMsgs, 0),
		homePath:           homePath,
	}
	return ch
}

func (ch *Child) Initialize(host hostNode, da executortypes.DANode, bridgeInfo opchildtypes.BridgeInfo) error {
	ch.host = host
	ch.bridgeInfo = bridgeInfo

	res, err := ch.host.QueryBatchInfos()
	if err != nil {
		return err
	}
	ch.batchInfos = res.BatchInfos

	ch.da = da
	if !ch.da.HasKey() {
		return errors.New("da has no key")
	}

	ch.batchFile, err = os.OpenFile(ch.homePath+"/batch", os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return err
	}

	ch.registerHandlers()
	return nil
}

func (ch *Child) Start(ctx context.Context) {
	ch.node.Start(ctx)
}

func (ch *Child) registerHandlers() {
	ch.node.RegisterBeginBlockHandler(ch.beginBlockHandler)
	ch.node.RegisterEventHandler(opchildtypes.EventTypeFinalizeTokenDeposit, ch.finalizeDepositHandler)
	ch.node.RegisterEventHandler(opchildtypes.EventTypeUpdateOracle, ch.updateOracleHandler)
	ch.node.RegisterEventHandler(opchildtypes.EventTypeInitiateTokenWithdrawal, ch.initiateWithdrawalHandler)
	ch.node.RegisterEndBlockHandler(ch.endBlockHandler)
}

func (ch Child) BroadcastMsgs(msgs nodetypes.ProcessedMsgs) {
	if !ch.node.HasKey() {
		return
	}
	ch.node.BroadcastMsgs(msgs)
}

func (ch Child) RawKVProcessedData(msgs []nodetypes.ProcessedMsgs, delete bool) ([]types.KV, error) {
	return ch.node.RawKVProcessedData(msgs, delete)
}

func (ch Child) BridgeId() uint64 {
	return ch.bridgeInfo.BridgeId
}
func (ch Child) AccountCodec() address.Codec {
	return ch.ac
}

func (ch Child) HasKey() bool {
	return ch.node.HasKey()
}

func (ch *Child) SetBridgeInfo(bridgeInfo opchildtypes.BridgeInfo) {
	ch.bridgeInfo = bridgeInfo
}
