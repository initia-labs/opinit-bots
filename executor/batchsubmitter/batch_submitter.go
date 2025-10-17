package batchsubmitter

import (
	"compress/gzip"
	"context"
	"io"
	"os"
	"sync"

	"go.uber.org/zap"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	"github.com/initia-labs/opinit-bots/node"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	childprovider "github.com/initia-labs/opinit-bots/provider/child"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"
)

type hostNode interface {
	ChainId() string
	QueryBatchInfos(types.Context, uint64) ([]ophosttypes.BatchInfoWithOutput, error)
	QueryBlock(ctx context.Context, height int64) (*coretypes.ResultBlock, error)
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
	localBatchInfo, err := GetLocalBatchInfo(bs.DB())
	if err != nil {
		return errors.Wrap(err, "failed to get local batch info")
	}
	bs.localBatchInfo = &localBatchInfo

	fileFlag := os.O_CREATE | os.O_RDWR | os.O_APPEND
	bs.batchFile, err = os.OpenFile(ctx.HomePath()+"/batch", fileFlag, 0640)
	if err != nil {
		return errors.Wrap(err, "failed to open batch file")
	}

	resetBatchFile := func(height int64) error {
		bs.localBatchInfo.Start = height
		bs.localBatchInfo.End = 0
		bs.localBatchInfo.BatchSize = 0

		err = SaveLocalBatchInfo(bs.DB(), *bs.localBatchInfo)
		if err != nil {
			return errors.Wrap(err, "failed to save local batch info")
		}
		// reset batch file
		return bs.emptyBatchFile()
	}

	if bs.node.GetSyncedHeight() == 0 { // HeightInitialized
		err = resetBatchFile(syncedHeight + 1)
		if err != nil {
			return err
		}
	} else {
		corrupted, err := bs.checkBatchFileCorruption()
		if err != nil {
			return err
		} else if corrupted {
			ctx.Logger().Warn("batch file is corrupted, resetting batch file", zap.Int64("start_height", bs.localBatchInfo.Start))
			err = resetBatchFile(bs.localBatchInfo.Start)
			if err != nil {
				return err
			}
			syncedHeight = bs.localBatchInfo.Start - 1
		}
	}
	// linux command gzip use level 6 as default
	bs.batchWriter, err = gzip.NewWriterLevel(bs.batchFile, 6)
	if err != nil {
		return errors.Wrap(err, "failed to create gzip writer")
	}

	err = bs.node.Initialize(ctx, syncedHeight, nil)
	if err != nil {
		return errors.Wrap(err, "failed to initialize node")
	}
	bs.host = host
	bs.bridgeInfo = bridgeInfo

	res, err := bs.host.QueryBatchInfos(ctx, bridgeInfo.BridgeId)
	if err != nil {
		return errors.Wrap(err, "failed to query batch infos")
	}
	bs.batchInfos = res
	if len(bs.batchInfos) == 0 {
		return errors.New("no batch info")
	}

	for i, batchInfo := range bs.batchInfos {
		if batchInfo.Output.L2BlockNumber != 0 && types.MustUint64ToInt64(batchInfo.Output.L2BlockNumber+1) > bs.node.GetHeight() {
			break
		} else if i > 0 {
			// dequeue the previous batch info
			bs.DequeueBatchInfo()
		}
	}

	bs.node.RegisterRawBlockHandler(bs.rawBlockHandler)
	return bs.LoadInternalStatus()
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

func (bs *BatchSubmitter) checkBatchFileCorruption() (bool, error) {
	info, err := bs.batchFile.Stat()
	if err != nil {
		return false, errors.Wrap(err, "failed to stat batch file")
	}
	if info.Size() == 0 {
		return false, nil
	}

	if _, err := bs.batchFile.Seek(0, io.SeekStart); err != nil {
		return false, errors.Wrap(err, "failed to seek batch file")
	}

	reader, err := gzip.NewReader(bs.batchFile)
	if err != nil {
		return true, nil
	}
	defer reader.Close()

	// #nosec G110
	if _, err := io.Copy(io.Discard, reader); err != nil {
		return true, nil
	}

	if _, err := bs.batchFile.Seek(0, io.SeekEnd); err != nil {
		return false, errors.Wrap(err, "failed to seek batch file")
	}
	return false, nil
}
