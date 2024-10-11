package batch

import (
	"compress/gzip"
	"context"
	"errors"
	"os"
	"sync"

	"go.uber.org/zap"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	"github.com/initia-labs/opinit-bots/node"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	childprovider "github.com/initia-labs/opinit-bots/provider/child"
	"github.com/initia-labs/opinit-bots/types"
)

type hostNode interface {
	QueryBatchInfos(context.Context, uint64) (*ophosttypes.QueryBatchInfosResponse, error)
}

type BatchSubmitter struct {
	version uint8

	node *node.Node
	host hostNode
	da   executortypes.DANode

	bridgeInfo opchildtypes.BridgeInfo

	cfg      nodetypes.NodeConfig
	batchCfg executortypes.BatchConfig
	db       types.DB
	logger   *zap.Logger

	opchildQueryClient opchildtypes.QueryClient

	batchInfoMu    *sync.Mutex
	batchInfos     []ophosttypes.BatchInfoWithOutput
	batchWriter    *gzip.Writer
	batchFile      *os.File
	localBatchInfo *executortypes.LocalBatchInfo

	processedMsgs []btypes.ProcessedMsgs

	chainID  string
	homePath string

	// status info
	LastBatchEndBlockNumber int64
}

func NewBatchSubmitterV1(
	cfg nodetypes.NodeConfig,
	batchCfg executortypes.BatchConfig,
	db types.DB, logger *zap.Logger,
	chainID, homePath, bech32Prefix string,
) *BatchSubmitter {
	appCodec, txConfig, err := childprovider.GetCodec(bech32Prefix)
	if err != nil {
		panic(err)
	}

	cfg.BroadcasterConfig = nil
	cfg.ProcessType = nodetypes.PROCESS_TYPE_RAW
	node, err := node.NewNode(cfg, db, logger, appCodec, txConfig)
	if err != nil {
		panic(err)
	}

	ch := &BatchSubmitter{
		version: 1,

		node: node,

		cfg:      cfg,
		batchCfg: batchCfg,

		db:     db,
		logger: logger,

		opchildQueryClient: opchildtypes.NewQueryClient(node.GetRPCClient()),

		batchInfoMu:    &sync.Mutex{},
		localBatchInfo: &executortypes.LocalBatchInfo{},

		processedMsgs: make([]btypes.ProcessedMsgs, 0),
		homePath:      homePath,
		chainID:       chainID,
	}
	return ch
}

func (bs *BatchSubmitter) Initialize(ctx context.Context, processedHeight int64, host hostNode, bridgeInfo opchildtypes.BridgeInfo) error {
	err := bs.node.Initialize(ctx, processedHeight)
	if err != nil {
		return err
	}
	bs.host = host
	bs.bridgeInfo = bridgeInfo

	res, err := bs.host.QueryBatchInfos(ctx, bridgeInfo.BridgeId)
	if err != nil {
		return err
	}
	bs.batchInfos = res.BatchInfos
	if len(bs.batchInfos) == 0 {
		return errors.New("no batch info")
	}
	for _, batchInfo := range bs.batchInfos {
		if len(bs.batchInfos) == 1 || types.MustUint64ToInt64(batchInfo.Output.L2BlockNumber+1) >= bs.node.GetHeight() {
			break
		}
		bs.DequeueBatchInfo()
	}

	fileFlag := os.O_CREATE | os.O_RDWR | os.O_APPEND
	bs.batchFile, err = os.OpenFile(bs.homePath+"/batch", fileFlag, 0640)
	if err != nil {
		return err
	}

	if bs.node.HeightInitialized() {
		bs.localBatchInfo.Start = bs.node.GetHeight()
		bs.localBatchInfo.End = 0
		bs.localBatchInfo.BatchFileSize = 0

		err = bs.saveLocalBatchInfo()
		if err != nil {
			return err
		}
		// reset batch file
		err := bs.batchFile.Truncate(0)
		if err != nil {
			return err
		}
		_, err = bs.batchFile.Seek(0, 0)
		if err != nil {
			return err
		}
	}
	// linux command gzip use level 6 as default
	bs.batchWriter, err = gzip.NewWriterLevel(bs.batchFile, 6)
	if err != nil {
		return err
	}

	bs.node.RegisterRawBlockHandler(bs.rawBlockHandler)
	return nil
}

func (bs *BatchSubmitter) SetDANode(da executortypes.DANode) {
	bs.da = da
}

func (bs *BatchSubmitter) Start(ctx context.Context) {
	bs.logger.Info("batch start", zap.Int64("height", bs.node.GetHeight()))
	bs.node.Start(ctx)
}

func (bs *BatchSubmitter) Close() {
	if bs.batchWriter != nil {
		bs.batchWriter.Close()
	}
	if bs.batchFile != nil {
		bs.batchFile.Close()
	}
}

func (bs *BatchSubmitter) SetBridgeInfo(bridgeInfo opchildtypes.BridgeInfo) {
	bs.bridgeInfo = bridgeInfo
}

func (bs *BatchSubmitter) ChainID() string {
	return bs.chainID
}

func (bs *BatchSubmitter) DA() executortypes.DANode {
	return bs.da
}

func (bs BatchSubmitter) Node() *node.Node {
	return bs.node
}
