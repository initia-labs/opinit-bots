package batch

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/cosmos/gogoproto/proto"

	sdk "github.com/cosmos/cosmos-sdk/types"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	executortypes "github.com/initia-labs/opinit-bots-go/executor/types"
	btypes "github.com/initia-labs/opinit-bots-go/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
	"github.com/initia-labs/opinit-bots-go/types"
)

func (bs *BatchSubmitter) rawBlockHandler(ctx context.Context, args nodetypes.RawBlockArgs) error {
	if len(bs.processedMsgs) != 0 {
		panic("must not happen, msgQueue should be empty")
	}

	pbb := new(cmtproto.Block)
	err := proto.Unmarshal(args.BlockBytes, pbb)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal block")
	}

	err = bs.prepareBatch(ctx, args.BlockHeight, pbb.Header.Time)
	if err != nil {
		return errors.Wrap(err, "failed to prepare batch")
	}

	blockBytes, err := bs.emptyOracleData(pbb)
	if err != nil {
		return err
	}

	err = bs.handleBatch(blockBytes)
	if err != nil {
		return errors.Wrap(err, "failed to handle batch")
	}

	err = bs.checkBatch(args.BlockHeight, pbb.Header.Time)
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
	if len(batchMsgKVs) > 0 {
		batchKVs = append(batchKVs, bs.SubmissionInfoToRawKV(pbb.Header.Time.UnixNano()))
	}
	err = bs.db.RawBatchSet(batchKVs...)
	if err != nil {
		return errors.Wrap(err, "failed to set raw batch")
	}
	// broadcast processed messages
	for _, processedMsg := range bs.processedMsgs {
		bs.da.BroadcastMsgs(processedMsg)
	}

	// clear processed messages
	bs.processedMsgs = bs.processedMsgs[:0]
	return nil
}

func (bs *BatchSubmitter) prepareBatch(ctx context.Context, blockHeight uint64, blockTime time.Time) error {
	// check whether the requested block height is reached to the l2 block number of the next batch info.
	if nextBatchInfo := bs.NextBatchInfo(); nextBatchInfo != nil && nextBatchInfo.Output.L2BlockNumber < blockHeight {
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

		// save sync info
		err = bs.node.SaveSyncInfo(nextBatchInfo.Output.L2BlockNumber)
		if err != nil {
			return errors.Wrap(err, "failed to save sync info")
		}

		// set last processed block height to l2 block number
		bs.node.SetSyncInfo(nextBatchInfo.Output.L2BlockNumber)
		bs.DequeueBatchInfo()

		// error will restart block process from nextBatchInfo.Output.L2BlockNumber + 1
		panic(fmt.Errorf("batch info updated: reset from %d", nextBatchInfo.Output.L2BlockNumber))
	}

	if bs.batchHeader != nil {
		// if the batch header end is not set, it means the batch is not finalized yet.
		if bs.batchHeader.End == 0 {
			return nil
		}

		err := bs.finalizeBatch(ctx, blockHeight)
		if err != nil {
			return errors.Wrap(err, "failed to finalize batch")
		}

		// update last submission time
		bs.lastSubmissionTime = blockTime
		bs.LastBatchEndBlockNumber = blockHeight
	}

	// reset batch header
	var err error
	bs.batchHeader = &executortypes.BatchHeader{}

	// linux command gzip use level 6 as default
	bs.batchWriter, err = gzip.NewWriterLevel(bs.batchFile, 6)
	if err != nil {
		return err
	}

	return nil
}

// write block bytes to batch file
func (bs *BatchSubmitter) handleBatch(blockBytes []byte) error {
	_, err := bs.batchWriter.Write(prependLength(blockBytes))
	if err != nil {
		return err
	}
	return nil
}

// finalize batch and create batch messages
func (bs *BatchSubmitter) finalizeBatch(ctx context.Context, blockHeight uint64) error {

	// write last block's commit to batch file
	rawCommit, err := bs.node.GetRPCClient().QueryRawCommit(ctx, int64(blockHeight))
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

	batchBuffer := make([]byte, bs.batchCfg.MaxChunkSize)
	checksums := make([][]byte, 0)

	// room for batch header
	bs.processedMsgs = append(bs.processedMsgs, btypes.ProcessedMsgs{
		Timestamp: time.Now().UnixNano(),
		Save:      true,
	})

	for offset := int64(0); ; offset += bs.batchCfg.MaxChunkSize {
		readLength, err := bs.batchFile.ReadAt(batchBuffer, offset)
		if err != nil && err != io.EOF {
			return err
		} else if readLength == 0 {
			break
		}

		// trim the buffer to the actual read length
		batchBuffer := batchBuffer[:readLength]
		msg, err := bs.da.CreateBatchMsg(batchBuffer)
		if err != nil {
			return err
		}
		bs.processedMsgs = append(bs.processedMsgs, btypes.ProcessedMsgs{
			Msgs:      []sdk.Msg{msg},
			Timestamp: time.Now().UnixNano(),
			Save:      true,
		})
		checksum := sha256.Sum256(batchBuffer)
		checksums = append(checksums, checksum[:])
		if int64(readLength) < bs.batchCfg.MaxChunkSize {
			break
		}
	}

	// update batch header
	bs.batchHeader.Chunks = checksums
	headerBytes, err := json.Marshal(bs.batchHeader)
	if err != nil {
		return err
	}
	msg, err := bs.da.CreateBatchMsg(headerBytes)
	if err != nil {
		return err
	}
	bs.processedMsgs[0].Msgs = []sdk.Msg{msg}

	// reset batch file
	err = bs.batchFile.Truncate(0)
	if err != nil {
		return err
	}
	_, err = bs.batchFile.Seek(0, 0)
	if err != nil {
		return err
	}

	bs.logger.Info("finalize batch",
		zap.Uint64("height", blockHeight),
		zap.Uint64("batch end", bs.batchHeader.End),
		zap.Int("chunks", len(checksums)),
		zap.Int("txs", len(bs.processedMsgs)),
	)
	return nil
}

func (bs *BatchSubmitter) checkBatch(blockHeight uint64, blockTime time.Time) error {
	fileSize, err := bs.batchFileSize()
	if err != nil {
		return err
	}

	// if the block time is after the last submission time + submission interval * 2/3
	// or the block time is after the last submission time + max submission time
	// or the batch file size is greater than (max chunks - 1) * max chunk size
	// then finalize the batch
	if blockTime.After(bs.lastSubmissionTime.Add(bs.bridgeInfo.BridgeConfig.SubmissionInterval*2/3)) ||
		blockTime.After(bs.lastSubmissionTime.Add(time.Duration(bs.batchCfg.MaxSubmissionTime)*time.Second)) ||
		fileSize > (bs.batchCfg.MaxChunks-1)*bs.batchCfg.MaxChunkSize {

		// finalize the batch
		bs.batchHeader.End = blockHeight
	}

	return nil
}

func (bs *BatchSubmitter) batchFileSize() (int64, error) {
	if bs.batchFile == nil {
		return 0, errors.New("batch file is not initialized")
	}
	info, err := bs.batchFile.Stat()
	if err != nil {
		return 0, errors.Wrap(err, "failed to get batch file stat")
	}
	return info.Size(), nil
}

// UpdateBatchInfo appends the batch info with the given chain, submitter, output index, and l2 block number
func (bs *BatchSubmitter) UpdateBatchInfo(chain string, submitter string, outputIndex uint64, l2BlockNumber uint64) {
	bs.batchInfoMu.Lock()
	defer bs.batchInfoMu.Unlock()

	// check if the batch info is already updated
	if bs.batchInfos[len(bs.batchInfos)-1].Output.L2BlockNumber >= l2BlockNumber {
		return
	}

	bs.batchInfos = append(bs.batchInfos, ophosttypes.BatchInfoWithOutput{
		BatchInfo: ophosttypes.BatchInfo{
			ChainType: ophosttypes.BatchInfo_ChainType(ophosttypes.BatchInfo_ChainType_value["CHAIN_TYPE_"+chain]),
			Submitter: submitter,
		},
		Output: ophosttypes.Output{
			L2BlockNumber: l2BlockNumber,
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
