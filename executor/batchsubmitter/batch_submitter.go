package batchsubmitter

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
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
	bs.batchInfos = res
	if len(bs.batchInfos) == 0 {
		return errors.New("no batch info")
	}

	localBatchInfo, err := GetLocalBatchInfo(bs.DB())
	if err != nil {
		return errors.Wrap(err, "failed to get local batch info")
	}
	bs.localBatchInfo = &localBatchInfo

	for i, batchInfo := range bs.batchInfos {
		if batchInfo.Output.L2BlockNumber != 0 && types.MustUint64ToInt64(batchInfo.Output.L2BlockNumber+1) > bs.node.GetHeight() {
			break
		} else if i > 0 {
			// dequeue the previous batch info
			bs.DequeueBatchInfo()
		}
	}

	fileFlag := os.O_CREATE | os.O_RDWR | os.O_APPEND
	bs.batchFile, err = os.OpenFile(ctx.HomePath()+"/batch", fileFlag, 0640)
	if err != nil {
		return errors.Wrap(err, "failed to open batch file")
	}

	var recoveredBlocks [][]byte
	if bs.node.HeightInitialized() {
		bs.localBatchInfo.Start = bs.node.GetHeight()
		bs.localBatchInfo.End = 0
		bs.localBatchInfo.BatchSize = 0

		err = SaveLocalBatchInfo(bs.DB(), *bs.localBatchInfo)
		if err != nil {
			return errors.Wrap(err, "failed to save local batch info")
		}
		// reset batch file
		err = bs.emptyBatchFile()
		if err != nil {
			return errors.Wrap(err, "failed to empty batch file")
		}
	} else {
		recoveredBlocks, err = bs.recoverIncompleteBatch(ctx)
		if err != nil {
			return errors.Wrap(err, "failed to recover incomplete batch")
		}
	}
	// linux command gzip use level 6 as default
	bs.batchWriter, err = gzip.NewWriterLevel(bs.batchFile, 6)
	if err != nil {
		return errors.Wrap(err, "failed to create gzip writer")
	}
	fileInfo, statErr := bs.batchFile.Stat()
	if statErr != nil {
		return errors.Wrap(statErr, "failed to stat batch file")
	}
	if len(recoveredBlocks) == 0 {
		bs.localBatchInfo.BatchSize = fileInfo.Size()
		if err := SaveLocalBatchInfo(bs.DB(), *bs.localBatchInfo); err != nil {
			return errors.Wrap(err, "failed to persist batch info")
		}
	}
	if len(recoveredBlocks) > 0 {
		for _, block := range recoveredBlocks {
			if _, err := bs.batchWriter.Write(prependLength(block)); err != nil {
				return errors.Wrap(err, "failed to rewrite recovered batch data")
			}
		}
		if err := bs.batchWriter.Flush(); err != nil {
			return errors.Wrap(err, "failed to flush recovered batch data")
		}
		fileInfo, statErr = bs.batchFile.Stat()
		if statErr != nil {
			return errors.Wrap(statErr, "failed to stat batch file after recovery")
		}
		bs.localBatchInfo.BatchSize = fileInfo.Size()
		if err := SaveLocalBatchInfo(bs.DB(), *bs.localBatchInfo); err != nil {
			return errors.Wrap(err, "failed to persist recovered batch info")
		}
		ctx.Logger().Warn("recovered incomplete batch data",
			zap.Int("recovered_blocks", len(recoveredBlocks)),
			zap.Int64("batch_start", bs.localBatchInfo.Start),
			zap.Int64("batch_size", bs.localBatchInfo.BatchSize),
		)
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

func (bs *BatchSubmitter) recoverIncompleteBatch(ctx types.Context) ([][]byte, error) {
	info, err := bs.batchFile.Stat()
	if err != nil {
		return nil, errors.Wrap(err, "failed to stat batch file")
	}
	if info.Size() == 0 {
		return nil, nil
	}

	if bs.localBatchInfo == nil {
		bs.localBatchInfo = &executortypes.LocalBatchInfo{}
	}

	if bs.localBatchInfo.End != 0 {
		ctx.Logger().Warn("found residual batch file after finalized batch; dropping",
			zap.Int64("batch_start", bs.localBatchInfo.Start),
			zap.Int64("batch_end", bs.localBatchInfo.End),
		)
		if err := bs.emptyBatchFile(); err != nil {
			return nil, errors.Wrap(err, "failed to empty batch file")
		}
		bs.localBatchInfo.BatchSize = 0
		return nil, nil
	}

	if _, err := bs.batchFile.Seek(0, 0); err != nil {
		return nil, errors.Wrap(err, "failed to seek batch file")
	}

	reader, err := gzip.NewReader(bs.batchFile)
	if err != nil {
		ctx.Logger().Warn("failed to open gzip reader for batch file; dropping file",
			zap.String("error", err.Error()))
		if err := bs.emptyBatchFile(); err != nil {
			return nil, errors.Wrap(err, "failed to empty batch file")
		}
		bs.localBatchInfo.BatchSize = 0
		return nil, nil
	}
	defer reader.Close()

	buf := new(bytes.Buffer)
	_, readErr := buf.ReadFrom(reader)

	data := buf.Bytes()
	blocks := make([][]byte, 0)
	partial := false
	for offset := 0; offset < len(data); {
		if len(data)-offset < 8 {
			partial = true
			break
		}
		length := binary.LittleEndian.Uint64(data[offset : offset+8])
		offset += 8

		if int(length) > len(data)-offset {
			partial = true
			break
		}

		block := make([]byte, int(length))
		copy(block, data[offset:offset+int(length)])
		blocks = append(blocks, block)
		offset += int(length)
	}

	if err := bs.emptyBatchFile(); err != nil {
		return nil, errors.Wrap(err, "failed to empty batch file")
	}

	if readErr != nil && !errors.Is(readErr, io.EOF) && !errors.Is(readErr, io.ErrUnexpectedEOF) {
		return blocks, errors.Wrap(readErr, "failed to read batch file")
	}

	if partial || errors.Is(readErr, io.ErrUnexpectedEOF) {
		ctx.Logger().Warn("recovered partial batch data",
			zap.Int("recovered_blocks", len(blocks)))
	}
	return blocks, nil
}
