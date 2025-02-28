package batchsubmitter

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	sdk "github.com/cosmos/cosmos-sdk/types"

	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	"github.com/initia-labs/opinit-bots/node"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	"github.com/initia-labs/opinit-bots/sentry_integration"
	"github.com/initia-labs/opinit-bots/types"
)

// prepareBatch prepares the batch info for the given block height.
// if there is more than one batch info in the queue and the start block height of the local batch info is greater than the l2 block number of the next batch info,
// it finalizes the current batch and occurs a panic to restart the block process from the next batch info's l2 block number + 1.
func (bs *BatchSubmitter) prepareBatch(ctx types.Context, blockHeight int64) error {
	localBatchInfo, err := GetLocalBatchInfo(bs.DB())
	if err != nil {
		return errors.Wrap(err, "failed to get local batch info")
	}
	bs.localBatchInfo = &localBatchInfo

	// check whether the requested block height is reached to the l2 block number of the next batch info.
	if nextBatchInfo := bs.NextBatchInfo(); nextBatchInfo != nil &&
		types.MustUint64ToInt64(nextBatchInfo.Output.L2BlockNumber) < bs.localBatchInfo.Start {
		// if the next batch info is reached, finalize the current batch and update the batch info.

		// wait until all processed msgs and pending txs are processed
		timer := time.NewTicker(1 * time.Second)
		defer timer.Stop()
		for {
			lenProcessedBatchMsgs, err := bs.da.LenProcessedBatchMsgs()
			if err != nil {
				return errors.Wrap(err, "failed to get processed msgs length")
			}
			lenPendingBatchTxs, err := bs.da.LenPendingBatchTxs()
			if err != nil {
				return errors.Wrap(err, "failed to get pending txs length")
			}
			if lenProcessedBatchMsgs == 0 && lenPendingBatchTxs == 0 {
				break
			}

			ctx.Logger().Info("waiting for processed batch msgs and pending batch txs to be processed before changing batch info",
				zap.Int("processed_batch_msgs", lenProcessedBatchMsgs),
				zap.Int("pending_batch_txs", lenPendingBatchTxs),
			)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
			}
		}

		if bs.batchWriter != nil {
			err := bs.batchWriter.Close()
			if err != nil {
				return errors.Wrap(err, "failed to close batch writer")
			}
		}
		err = bs.emptyBatchFile()
		if err != nil {
			return errors.Wrap(err, "failed to empty batch file")
		}

		bs.localBatchInfo.Start = types.MustUint64ToInt64(nextBatchInfo.Output.L2BlockNumber) + 1
		bs.localBatchInfo.End = 0
		bs.localBatchInfo.BatchSize = 0
		err = SaveLocalBatchInfo(bs.DB(), *bs.localBatchInfo)
		if err != nil {
			return errors.Wrap(err, "failed to save local batch info")
		}

		err = node.SetSyncedHeight(bs.DB(), types.MustUint64ToInt64(nextBatchInfo.Output.L2BlockNumber))
		if err != nil {
			return errors.Wrap(err, "failed to set synced height")
		}

		// error will restart block process from nextBatchInfo.Output.L2BlockNumber + 1
		panic(fmt.Errorf("batch info updated: reset from %d", nextBatchInfo.Output.L2BlockNumber))
	}

	// if the current batch is finalized, reset the batch file and batch info
	if bs.localBatchInfo.End != 0 {
		err = bs.emptyBatchFile()
		if err != nil {
			return errors.Wrap(err, "failed to empty batch file")
		}

		bs.localBatchInfo.Start = blockHeight
		bs.localBatchInfo.End = 0
		bs.localBatchInfo.BatchSize = 0
		bs.batchWriter.Reset(bs.batchFile)
	}
	return nil
}

// handleBatch writes the block bytes to the batch file.
func (bs *BatchSubmitter) handleBatch(blockBytes []byte) (int, error) {
	if len(blockBytes) == 0 {
		return 0, errors.New("block bytes is empty")
	}
	return bs.batchWriter.Write(prependLength(blockBytes))
}

// finalizeBatch finalizes the batch by writing the last block's commit to the batch file.
// it creates batch messages for the batch data and adds them to the processed messages.
// the batch data is divided into chunks and each chunk is added to the processed messages.
func (bs *BatchSubmitter) finalizeBatch(parentCtx types.Context, blockHeight int64) error {
	transaction, ctx := sentry_integration.StartSentryTransaction(parentCtx, "finalizeBatch", "Finalizes the batch")
	defer transaction.Finish()
	transaction.SetTag("height", fmt.Sprintf("%d", blockHeight))

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
		return errors.Wrap(err, "failed to get batch file size")
	}
	bs.localBatchInfo.BatchSize = fileSize

	batchBuffer := make([]byte, bs.batchCfg.MaxChunkSize)
	checksums := make([][]byte, 0)

	// TODO: improve this logic to avoid hold all the batch data in memory
	chunks := make([][]byte, 0)
	for offset := int64(0); ; {
		readLength, err := bs.batchFile.ReadAt(batchBuffer, offset)
		if err != nil && err != io.EOF {
			return errors.Wrap(err, "failed to read batch file")
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

	msg, sender, err := bs.da.CreateBatchMsg(headerData)
	if err != nil {
		return errors.Wrap(err, "failed to create batch msg")
	} else if msg != nil {
		bs.processedMsgs = append(bs.processedMsgs, btypes.ProcessedMsgs{
			Sender:    sender,
			Msgs:      []sdk.Msg{msg},
			Timestamp: types.CurrentNanoTimestamp(),
			Save:      true,
		})
	}

	for i, chunk := range chunks {
		chunkData := executortypes.MarshalBatchDataChunk(
			types.MustInt64ToUint64(bs.localBatchInfo.Start),
			types.MustInt64ToUint64(bs.localBatchInfo.End),
			types.MustInt64ToUint64(int64(i)),
			types.MustInt64ToUint64(int64(len(checksums))),
			chunk,
		)
		msg, sender, err := bs.da.CreateBatchMsg(chunkData)
		if err != nil {
			return errors.Wrap(err, "failed to create batch msg")
		} else if msg != nil {
			bs.processedMsgs = append(bs.processedMsgs, btypes.ProcessedMsgs{
				Sender:    sender,
				Msgs:      []sdk.Msg{msg},
				Timestamp: types.CurrentNanoTimestamp(),
				Save:      true,
			})
		}
	}

	ctx.Logger().Info("finalize batch",
		zap.Int64("height", blockHeight),
		zap.Int64("batch start", bs.localBatchInfo.Start),
		zap.Int64("batch end", bs.localBatchInfo.End),
		zap.Int64("batch size", bs.localBatchInfo.BatchSize),
		zap.Int("chunks", len(checksums)),
		zap.Int("txs", len(bs.processedMsgs)),
	)
	return nil
}

// checkBatch checks whether the batch should be finalized or not.
// if the block time is after the last submission time + submission interval * 2/3
// or the block time is after the last submission time + max submission time
// or the batch file size is greater than (max chunks - 1) * max chunk size
// then finalize the batch
func (bs *BatchSubmitter) checkBatch(blockHeight int64, latestHeight int64, blockTime time.Time) bool {
	if (blockHeight == latestHeight && blockTime.After(bs.localBatchInfo.LastSubmissionTime.Add(bs.bridgeInfo.BridgeConfig.SubmissionInterval*2/3))) ||
		(blockHeight == latestHeight && blockTime.After(bs.localBatchInfo.LastSubmissionTime.Add(time.Duration(bs.batchCfg.MaxSubmissionTime)*time.Second))) ||
		bs.localBatchInfo.BatchSize > (bs.batchCfg.MaxChunks-1)*bs.batchCfg.MaxChunkSize {
		return true
	}
	return false
}

// batchFileSize returns the size of the batch file.
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

func (bs *BatchSubmitter) emptyBatchFile() error {
	// reset batch file
	err := bs.batchFile.Truncate(0)
	if err != nil {
		return errors.Wrap(err, "failed to truncate batch file")
	}
	_, err = bs.batchFile.Seek(0, 0)
	if err != nil {
		return errors.Wrap(err, "failed to seek batch file")
	}
	return nil
}

// prependLength prepends the length of the data to the data.
func prependLength(data []byte) []byte {
	lengthBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(lengthBytes, uint64(len(data)))
	return append(lengthBytes, data...)
}

func (bs *BatchSubmitter) submitGenesis(ctx types.Context) error {
	res, err := bs.node.GetRPCClient().Genesis(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to query genesis")
	}

	genesisBz, err := json.Marshal(res.Genesis)
	if err != nil {
		return errors.Wrap(err, "failed to marshal genesis")
	}

	chunkLength := (len(genesisBz) + int(bs.batchCfg.MaxChunkSize) - 1) / int(bs.batchCfg.MaxChunkSize)
	for i := 0; i < chunkLength; i++ {
		start := i * int(bs.batchCfg.MaxChunkSize)
		end := start + int(bs.batchCfg.MaxChunkSize)
		if end > len(genesisBz) {
			end = len(genesisBz)
		}
		chunk := genesisBz[start:end]
		if len(chunk) == 0 {
			break
		}

		chunkData := executortypes.MarshalBatchDataGenesis(
			types.MustInt64ToUint64(int64(i)),
			types.MustInt64ToUint64(int64(chunkLength)),
			chunk,
		)
		msg, sender, err := bs.da.CreateBatchMsg(chunkData)
		if err != nil {
			return errors.Wrap(err, "failed to create batch msg")
		} else if msg != nil {
			bs.processedMsgs = append(bs.processedMsgs, btypes.ProcessedMsgs{
				Sender:    sender,
				Msgs:      []sdk.Msg{msg},
				Timestamp: types.CurrentNanoTimestamp(),
				Save:      true,
			})
		}
	}
	ctx.Logger().Info("submit genesis",
		zap.Int("genesis size", len(genesisBz)),
		zap.Int("chunk length", chunkLength),
	)
	return nil
}
