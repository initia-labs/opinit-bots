package batch

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	executortypes "github.com/initia-labs/opinit-bots-go/executor/types"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
	"github.com/initia-labs/opinit-bots-go/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/gogoproto/proto"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
)

func (bs *BatchSubmitter) rawBlockHandler(args nodetypes.RawBlockArgs) error {
	if len(bs.processedMsgs) != 0 {
		panic("must not happen, msgQueue should be empty")
	}

	pbb := new(cmtproto.Block)
	err := proto.Unmarshal(args.BlockBytes, pbb)
	if err != nil {
		return err
	}

	err = bs.prepareBatch(args.BlockHeight, pbb.Header.Time)
	if err != nil {
		return err
	}
	err = bs.handleBatch(args.BlockBytes)
	if err != nil {
		return err
	}

	err = bs.checkBatch(args.BlockHeight, pbb.Header.Time)
	if err != nil {
		return err
	}

	batchKVs := make([]types.RawKV, 0)
	batchKVs = append(batchKVs, bs.node.SyncInfoToRawKV(args.BlockHeight))
	batchMsgkvs, err := bs.da.ProcessedMsgsToRawKV(bs.processedMsgs, false)
	if err != nil {
		return err
	}
	batchKVs = append(batchKVs, batchMsgkvs...)
	if len(batchMsgkvs) > 0 {
		batchKVs = append(batchKVs, bs.SubmissionInfoToRawKV(pbb.Header.Time.UnixNano()))
	}
	err = bs.db.RawBatchSet(batchKVs...)
	if err != nil {
		return err
	}
	for _, processedMsg := range bs.processedMsgs {
		bs.da.BroadcastMsgs(processedMsg)
	}
	bs.processedMsgs = bs.processedMsgs[:0]
	return nil
}

func (bs *BatchSubmitter) prepareBatch(blockHeight uint64, blockTime time.Time) error {
	if nextBatchInfo := bs.NextBatchInfo(); nextBatchInfo != nil && nextBatchInfo.Output.L2BlockNumber < blockHeight {
		if bs.batchWriter != nil {
			err := bs.batchWriter.Close()
			if err != nil {
				return err
			}
		}
		err := bs.batchFile.Truncate(0)
		if err != nil {
			return err
		}
		_, err = bs.batchFile.Seek(0, 0)
		if err != nil {
			return err
		}
		err = bs.node.SaveSyncInfo(nextBatchInfo.Output.L2BlockNumber)
		if err != nil {
			return err
		}
		// set last processed block height to l2 block number
		bs.node.SetSyncInfo(nextBatchInfo.Output.L2BlockNumber)
		bs.PopBatchInfo()

		// error will restart block process from nextBatchInfo.Output.L2BlockNumber + 1
		return fmt.Errorf("batch info updated: reset from %d", nextBatchInfo.Output.L2BlockNumber)
	}

	if bs.batchHeader != nil {
		if bs.batchHeader.End == 0 {
			return nil
		}
		err := bs.finalizeBatch(blockHeight)
		if err != nil {
			return err
		}
		bs.lastSubmissionTime = blockTime
	}
	bs.batchHeader = &executortypes.BatchHeader{}
	var err error
	// linux command gzip use level 6 as default
	bs.batchWriter, err = gzip.NewWriterLevel(bs.batchFile, 6)
	if err != nil {
		return err
	}
	return nil
}

func (bs *BatchSubmitter) handleBatch(blockBytes []byte) error {
	encodedBlockBytes := base64.StdEncoding.EncodeToString(blockBytes)
	_, err := bs.batchWriter.Write(append([]byte(encodedBlockBytes), ','))
	if err != nil {
		return err
	}
	return nil
}

func (bs *BatchSubmitter) finalizeBatch(blockHeight uint64) error {
	rawCommit, err := bs.node.QueryRawCommit(int64(blockHeight))
	if err != nil {
		return err
	}
	encodedRawCommit := base64.StdEncoding.EncodeToString(rawCommit)
	_, err = bs.batchWriter.Write([]byte(encodedRawCommit))
	if err != nil {
		return err
	}
	err = bs.batchWriter.Close()
	if err != nil {
		return err
	}

	batchBuffer := make([]byte, bs.batchCfg.MaxChunkSize)
	checksums := make([][]byte, 0)
	// room for batch header
	bs.processedMsgs = append(bs.processedMsgs, nodetypes.ProcessedMsgs{
		Timestamp: time.Now().UnixNano(),
		Save:      true,
	})

	for offset := int64(0); ; offset += int64(bs.batchCfg.MaxChunkSize) {
		readLength, err := bs.batchFile.ReadAt(batchBuffer, offset)
		if err != nil && err != io.EOF {
			return err
		} else if readLength == 0 {
			break
		}
		batchBuffer = batchBuffer[:readLength]
		msg, err := bs.da.CreateBatchMsg(batchBuffer)
		if err != nil {
			return err
		}
		bs.processedMsgs = append(bs.processedMsgs, nodetypes.ProcessedMsgs{
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
	err = bs.batchFile.Truncate(0)
	if err != nil {
		return err
	}
	_, err = bs.batchFile.Seek(0, 0)
	if err != nil {
		return err
	}
	return nil
}

func (bs *BatchSubmitter) checkBatch(blockHeight uint64, blockTime time.Time) error {
	info, err := bs.batchFile.Stat()
	if err != nil {
		return err
	}

	if blockTime.After(bs.lastSubmissionTime.Add(bs.bridgeInfo.BridgeConfig.SubmissionInterval*2/3)) ||
		blockTime.After(bs.lastSubmissionTime.Add(time.Duration(bs.batchCfg.MaxSubmissionTime)*time.Second)) ||
		info.Size() > (bs.batchCfg.MaxChunks-1)*bs.batchCfg.MaxChunkSize {
		bs.batchHeader.End = blockHeight
	}
	return nil
}

func (bs *BatchSubmitter) UpdateBatchInfo(chain string, submitter string, outputIndex uint64, l2BlockNumber uint64) {
	bs.batchInfoMu.Lock()
	defer bs.batchInfoMu.Unlock()

	bs.batchInfos = append(bs.batchInfos, ophosttypes.BatchInfoWithOutput{
		BatchInfo: ophosttypes.BatchInfo{
			Chain:     chain,
			Submitter: submitter,
		},
		Output: ophosttypes.Output{
			L2BlockNumber: l2BlockNumber,
		},
	})
}

func (bs *BatchSubmitter) BatchInfo() *ophosttypes.BatchInfoWithOutput {
	bs.batchInfoMu.Lock()
	defer bs.batchInfoMu.Unlock()

	return &bs.batchInfos[0]
}

func (bs *BatchSubmitter) NextBatchInfo() *ophosttypes.BatchInfoWithOutput {
	bs.batchInfoMu.Lock()
	defer bs.batchInfoMu.Unlock()
	if len(bs.batchInfos) == 1 {
		return nil
	}
	return &bs.batchInfos[1]
}

func (bs *BatchSubmitter) PopBatchInfo() {
	bs.batchInfoMu.Lock()
	defer bs.batchInfoMu.Unlock()

	bs.batchInfos = bs.batchInfos[1:]
}
