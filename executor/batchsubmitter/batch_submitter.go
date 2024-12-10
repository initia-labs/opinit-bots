package batchsubmitter

import (
	"compress/gzip"
	"context"
	"os"
	"sync"

	"go.uber.org/zap"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	"github.com/initia-labs/opinit-bots/node"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	childprovider "github.com/initia-labs/opinit-bots/provider/child"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"
)

type hostNode interface {
	QueryBatchInfos(context.Context, uint64) (*ophosttypes.QueryBatchInfosResponse, error)
}

type BatchSubmitter struct {
	version uint8

	node *node.Node
	host hostNode
	da   executortypes.DANode

	bridgeInfo ophosttypes.QueryBridgeResponse

	cfg      nodetypes.NodeConfig
	batchCfg executortypes.BatchConfig

	batchInfoMu    *sync.Mutex
	batchInfos     []ophosttypes.BatchInfoWithOutput
	batchWriter    *gzip.Writer
	batchFile      *os.File
	localBatchInfo *executortypes.LocalBatchInfo

	processedMsgs []btypes.ProcessedMsgs

	chainID string

	stage types.CommitDB

	// status info
	LastBatchEndBlockNumber int64
}

func NewBatchSubmitterV1(
	cfg nodetypes.NodeConfig,
	batchCfg executortypes.BatchConfig,
	db types.DB,
	chainID string,
) *BatchSubmitter {
	appCodec, txConfig, err := childprovider.GetCodec(cfg.Bech32Prefix)
	if err != nil {
		panic(err)
	}

	cfg.BroadcasterConfig = nil
	cfg.ProcessType = nodetypes.PROCESS_TYPE_RAW
	node, err := node.NewNode(cfg, db, appCodec, txConfig)
	if err != nil {
		panic(errors.Wrap(err, "failed to create node"))
	}

	ch := &BatchSubmitter{
		version: 1,

		node: node,

		cfg:      cfg,
		batchCfg: batchCfg,

		batchInfoMu:    &sync.Mutex{},
		localBatchInfo: &executortypes.LocalBatchInfo{},

		processedMsgs: make([]btypes.ProcessedMsgs, 0),
		chainID:       chainID,

		stage: db.NewStage(),
	}
	return ch
}

func (bs *BatchSubmitter) Initialize(ctx types.Context, syncedHeight int64, host hostNode, bridgeInfo ophosttypes.QueryBridgeResponse) error {
	err := bs.node.Initialize(ctx, syncedHeight, nil)
	if err != nil {
		return errors.Wrap(err, "failed to initialize node")
	}
	bs.host = host
	bs.bridgeInfo = bridgeInfo

	res, err := bs.host.QueryBatchInfos(ctx, bridgeInfo.BridgeId)
	if err != nil {
		return errors.Wrap(err, "failed to query batch infos")
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
	bs.batchFile, err = os.OpenFile(ctx.HomePath()+"/batch", fileFlag, 0640)
	if err != nil {
		return errors.Wrap(err, "failed to open batch file")
	}

	if bs.node.HeightInitialized() {
		bs.localBatchInfo.Start = bs.node.GetHeight()
		bs.localBatchInfo.End = 0
		bs.localBatchInfo.BatchSize = 0

		err = SaveLocalBatchInfo(bs.DB(), *bs.localBatchInfo)
		if err != nil {
			return errors.Wrap(err, "failed to save local batch info")
		}
		// reset batch file
		err := bs.batchFile.Truncate(0)
		if err != nil {
			return errors.Wrap(err, "failed to truncate batch file")
		}
		_, err = bs.batchFile.Seek(0, 0)
		if err != nil {
			return errors.Wrap(err, "failed to seek batch file")
		}
	}
	// linux command gzip use level 6 as default
	bs.batchWriter, err = gzip.NewWriterLevel(bs.batchFile, 6)
	if err != nil {
		return errors.Wrap(err, "failed to create gzip writer")
	}

	bs.node.RegisterRawBlockHandler(bs.rawBlockHandler)
	return nil
}

func (bs *BatchSubmitter) SetDANode(da executortypes.DANode) {
	bs.da = da
}

func (bs *BatchSubmitter) Start(ctx types.Context) {
	ctx.Logger().Info("batch start", zap.Int64("height", bs.node.GetHeight()))
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

func (bs *BatchSubmitter) SetBridgeInfo(bridgeInfo ophosttypes.QueryBridgeResponse) {
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

func (bs BatchSubmitter) DB() types.DB {
	return bs.node.DB()
}
