package batch

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/cosmos/gogoproto/proto"

	sdk "github.com/cosmos/cosmos-sdk/types"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"
)

func (bs *BatchSubmitter) rawBlockHandler(ctx context.Context, args nodetypes.RawBlockArgs) error {
	// clear processed messages
	bs.processedMsgs = bs.processedMsgs[:0]

	pbb := new(cmtproto.Block)
	err := proto.Unmarshal(args.BlockBytes, pbb)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal block")
	}

	err = bs.prepareBatch(args.BlockHeight)
	if err != nil {
		return errors.Wrap(err, "failed to prepare batch")
	}

	blockBytes, err := bs.emptyOracleData(pbb)
	if err != nil {
		return err
	}

	_, err = bs.handleBatch(blockBytes)
	if err != nil {
		return errors.Wrap(err, "failed to handle batch")
	}

	err = bs.checkBatch(ctx, args.BlockHeight, args.LatestHeight, pbb.Header.Time)
	if err != nil {
		return errors.Wrap(err, "failed to check batch")
	}

	// store the processed state into db with batch operation
	batchKVs := make([]types.RawKV, 0)
	batchKVs = append(batchKVs, bs.node.SyncInfoToRawKV(args.BlockHeight))
	batchMsgKVs, err := bs.da.ProcessedMsgsToRawKV(bs.processedMsgs, false)
	if err != nil {
		return errors.Wrap(err, "failed to convert processed messages to raw key value")
	}
	batchKVs = append(batchKVs, batchMsgKVs...)

	kv, err := bs.localBatchInfoToRawKV()
	if err != nil {
		return err
	}
	batchKVs = append(batchKVs, kv)

	err = bs.db.RawBatchSet(batchKVs...)
	if err != nil {
		return errors.Wrap(err, "failed to set raw batch")
	}
	// broadcast processed messages
	for _, processedMsg := range bs.processedMsgs {
		bs.da.BroadcastMsgs(processedMsg)
	}
	return nil
}

func (bs *BatchSubmitter) prepareBatch(blockHeight int64) error {
	err := bs.loadLocalBatchInfo()
	if err != nil {
		return err
	}

	// check whether the requested block height is reached to the l2 block number of the next batch info.
	if nextBatchInfo := bs.NextBatchInfo(); nextBatchInfo != nil && types.MustUint64ToInt64(nextBatchInfo.Output.L2BlockNumber) < blockHeight {
		// if the next batch info is reached, finalize the current batch and update the batch info.
		if bs.batchWriter != nil {
			err := bs.batchWriter.Close()
			if err != nil {
				return errors.Wrap(err, "failed to close batch writer")
			}
		}
		err := bs.batchFile.Truncate(0)
		if err != nil {
			return errors.Wrap(err, "failed to truncate batch file")
		}
		_, err = bs.batchFile.Seek(0, 0)
		if err != nil {
			return errors.Wrap(err, "failed to seek batch file")
		}

		// save sync info
		err = bs.node.SaveSyncInfo(types.MustUint64ToInt64(nextBatchInfo.Output.L2BlockNumber))
		if err != nil {
			return errors.Wrap(err, "failed to save sync info")
		}
		bs.localBatchInfo.Start = types.MustUint64ToInt64(nextBatchInfo.Output.L2BlockNumber) + 1
		bs.localBatchInfo.End = 0
		bs.localBatchInfo.BatchFileSize = 0
		err = bs.saveLocalBatchInfo()
		if err != nil {
			return err
		}
		// set last processed block height to l2 block number
		bs.node.SetSyncInfo(types.MustUint64ToInt64(nextBatchInfo.Output.L2BlockNumber))
		bs.DequeueBatchInfo()

		// error will restart block process from nextBatchInfo.Output.L2BlockNumber + 1
		panic(fmt.Errorf("batch info updated: reset from %d", nextBatchInfo.Output.L2BlockNumber))
	}

	if bs.localBatchInfo.End != 0 {
		// reset batch file
		err := bs.batchFile.Truncate(0)
		if err != nil {
			return err
		}
		_, err = bs.batchFile.Seek(0, 0)
		if err != nil {
			return err
		}

		bs.localBatchInfo.BatchFileSize = 0
		bs.localBatchInfo.Start = blockHeight
		bs.localBatchInfo.End = 0

		bs.batchWriter.Reset(bs.batchFile)
	}
	return nil
}

// write block bytes to batch file
func (bs *BatchSubmitter) handleBatch(blockBytes []byte) (int, error) {
	return bs.batchWriter.Write(prependLength(blockBytes))
}

// finalize batch and create batch messages
func (bs *BatchSubmitter) finalizeBatch(ctx context.Context, blockHeight int64) error {
	// write last block's commit to batch file
	rawCommit, err := bs.node.GetRPCClient().QueryRawCommit(ctx, blockHeight)
	if err != nil {
		return errors.Wrap(err, "failed to query raw commit")
	}
	_, err = bs.batchWriter.Write(prependLength(rawCommit))
	if err != nil {
		return errors.Wrap(err, "failed to write raw commit")
	}
	err = bs.batchWriter.Close()
	if err != nil {
		return errors.Wrap(err, "failed to close batch writer")
	}
	fileSize, err := bs.batchFileSize(false)
	if err != nil {
		return err
	}
	bs.localBatchInfo.BatchFileSize = fileSize

	batchBuffer := make([]byte, bs.batchCfg.MaxChunkSize)
	checksums := make([][]byte, 0)

	// TODO: improve this logic to avoid hold all the batch data in memory
	chunks := make([][]byte, 0)
	for offset := int64(0); ; {
		readLength, err := bs.batchFile.ReadAt(batchBuffer, offset)
		if err != nil && err != io.EOF {
			return err
		} else if readLength == 0 {
			break
		}

		// trim the buffer to the actual read length
		chunk := bytes.Clone(batchBuffer[:readLength])
		chunks = append(chunks, chunk)

		checksum := executortypes.GetChecksumFromChunk(chunk)
		checksums = append(checksums, checksum[:])
		if int64(readLength) < bs.batchCfg.MaxChunkSize {
			break
		}
		offset += int64(readLength)
	}

	headerData := executortypes.MarshalBatchDataHeader(
		types.MustInt64ToUint64(bs.localBatchInfo.Start),
		types.MustInt64ToUint64(bs.localBatchInfo.End),
		checksums,
	)

	msg, err := bs.da.CreateBatchMsg(headerData)
	if err != nil {
		return err
	}
	bs.processedMsgs = append(bs.processedMsgs, btypes.ProcessedMsgs{
		Msgs:      []sdk.Msg{msg},
		Timestamp: time.Now().UnixNano(),
		Save:      true,
	})

	for i, chunk := range chunks {
		chunkData := executortypes.MarshalBatchDataChunk(
			types.MustInt64ToUint64(bs.localBatchInfo.Start),
			types.MustInt64ToUint64(bs.localBatchInfo.End),
			types.MustInt64ToUint64(int64(i)),
			types.MustInt64ToUint64(int64(len(checksums))),
			chunk,
		)
		msg, err := bs.da.CreateBatchMsg(chunkData)
		if err != nil {
			return err
		}
		bs.processedMsgs = append(bs.processedMsgs, btypes.ProcessedMsgs{
			Msgs:      []sdk.Msg{msg},
			Timestamp: time.Now().UnixNano(),
			Save:      true,
		})
	}

	bs.logger.Info("finalize batch",
		zap.Int64("height", blockHeight),
		zap.Int64("batch start", bs.localBatchInfo.Start),
		zap.Int64("batch end", bs.localBatchInfo.End),
		zap.Int64("batch file size ", bs.localBatchInfo.BatchFileSize),
		zap.Int("chunks", len(checksums)),
		zap.Int("txs", len(bs.processedMsgs)),
	)
	return nil
}

func (bs *BatchSubmitter) checkBatch(ctx context.Context, blockHeight int64, latestHeight int64, blockTime time.Time) error {
	fileSize, err := bs.batchFileSize(true)
	if err != nil {
		return err
	}

	bs.localBatchInfo.BatchFileSize = fileSize
	// if the block time is after the last submission time + submission interval * 2/3
	// or the block time is after the last submission time + max submission time
	// or the batch file size is greater than (max chunks - 1) * max chunk size
	// then finalize the batch
	if (blockHeight == latestHeight && blockTime.After(bs.localBatchInfo.LastSubmissionTime.Add(bs.bridgeInfo.BridgeConfig.SubmissionInterval*2/3))) ||
		(blockHeight == latestHeight && blockTime.After(bs.localBatchInfo.LastSubmissionTime.Add(time.Duration(bs.batchCfg.MaxSubmissionTime)*time.Second))) ||
		fileSize > (bs.batchCfg.MaxChunks-1)*bs.batchCfg.MaxChunkSize {

		// finalize the batch
		bs.LastBatchEndBlockNumber = blockHeight
		bs.localBatchInfo.LastSubmissionTime = blockTime
		bs.localBatchInfo.End = blockHeight

		err := bs.finalizeBatch(ctx, blockHeight)
		if err != nil {
			return errors.Wrap(err, "failed to finalize batch")
		}
	}
	return nil
}

func (bs *BatchSubmitter) batchFileSize(flush bool) (int64, error) {
	if bs.batchFile == nil {
		return 0, errors.New("batch file is not initialized")
	}
	if flush {
		err := bs.batchWriter.Flush()
		if err != nil {
			return 0, errors.Wrap(err, "failed to flush batch writer")
		}
	}

	info, err := bs.batchFile.Stat()
	if err != nil {
		return 0, errors.Wrap(err, "failed to get batch file stat")
	}
	return info.Size(), nil
}

// UpdateBatchInfo appends the batch info with the given chain, submitter, output index, and l2 block number
func (bs *BatchSubmitter) UpdateBatchInfo(chain string, submitter string, outputIndex uint64, l2BlockNumber int64) {
	bs.batchInfoMu.Lock()
	defer bs.batchInfoMu.Unlock()

	// check if the batch info is already updated
	if types.MustUint64ToInt64(bs.batchInfos[len(bs.batchInfos)-1].Output.L2BlockNumber) >= l2BlockNumber {
		return
	}

	bs.batchInfos = append(bs.batchInfos, ophosttypes.BatchInfoWithOutput{
		BatchInfo: ophosttypes.BatchInfo{
			ChainType: ophosttypes.BatchInfo_ChainType(ophosttypes.BatchInfo_ChainType_value["CHAIN_TYPE_"+chain]),
			Submitter: submitter,
		},
		Output: ophosttypes.Output{
			L2BlockNumber: types.MustInt64ToUint64(l2BlockNumber),
		},
	})
}

// BatchInfo returns the current batch info
func (bs *BatchSubmitter) BatchInfo() *ophosttypes.BatchInfoWithOutput {
	bs.batchInfoMu.Lock()
	defer bs.batchInfoMu.Unlock()

	return &bs.batchInfos[0]
}

// NextBatchInfo returns the next batch info in the queue
func (bs *BatchSubmitter) NextBatchInfo() *ophosttypes.BatchInfoWithOutput {
	bs.batchInfoMu.Lock()
	defer bs.batchInfoMu.Unlock()
	if len(bs.batchInfos) == 1 {
		return nil
	}
	return &bs.batchInfos[1]
}

// DequeueBatchInfo removes the first batch info from the queue
func (bs *BatchSubmitter) DequeueBatchInfo() {
	bs.batchInfoMu.Lock()
	defer bs.batchInfoMu.Unlock()

	bs.batchInfos = bs.batchInfos[1:]
}
