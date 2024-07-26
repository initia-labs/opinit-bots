package child

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"time"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	executortypes "github.com/initia-labs/opinit-bots-go/executor/types"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
)

func (ch *Child) prepareBatch(blockHeight uint64) error {
	if ch.batchWriter != nil {
		return nil
	}
	var err error

	ch.batchFile, err = os.OpenFile("batch", os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return err
	}

	// linux command gzip use level 6 as default
	ch.batchWriter, err = gzip.NewWriterLevel(ch.batchFile, 6)
	if err != nil {
		return err
	}
	ch.batchHeader = executortypes.BatchHeader{
		Start: blockHeight,
	}
	return nil
}

func (ch *Child) handleBatch(block *cmtproto.Block) error {
	// we are syncing
	if ch.batchHeader.End != 0 {
		return nil
	}
	blockBytes, err := block.Marshal()
	if err != nil {
		return err
	}

	encodedBlockBytes := base64.StdEncoding.EncodeToString(blockBytes)
	_, err = ch.batchWriter.Write(append([]byte(encodedBlockBytes), ','))
	if err != nil {
		return err
	}
	return nil
}

func (ch *Child) finalizeBatch(blockHeight uint64) error {
	// we are syncing
	if ch.batchHeader.End != 0 {
		return nil
	}

	rawCommit, err := ch.node.QueryRawCommit(int64(blockHeight))
	if err != nil {
		return err
	}
	encodedRawCommit := base64.StdEncoding.EncodeToString(rawCommit)
	_, err = ch.batchWriter.Write([]byte(encodedRawCommit))
	if err != nil {
		return err
	}
	err = ch.batchWriter.Close()
	if err != nil {
		return err
	}

	batchBuffer := make([]byte, ch.batchCfg.MaxBatchSize)
	readLength := 0

	checksums := make([][]byte, 0)
	// room for batch header
	ch.processedMsgs = append(ch.processedMsgs, nodetypes.ProcessedMsgs{
		Timestamp: time.Now().UnixNano(),
		Save:      true,
	})

	for offset := int64(0); int64(readLength) == ch.batchCfg.MaxBatchSize; offset += int64(ch.batchCfg.MaxBatchSize) {
		readLength, err := ch.batchFile.ReadAt(batchBuffer, offset)
		if err == io.EOF || readLength == 0 {
			break
		} else if err != nil {
			return err
		}
		batchBuffer = batchBuffer[:readLength]

		msg, err := ch.createBatchMsg(batchBuffer)
		if err != nil {
			return err
		}

		ch.processedMsgs = append(ch.processedMsgs, nodetypes.ProcessedMsgs{
			Msgs:      []sdk.Msg{msg},
			Timestamp: time.Now().UnixNano(),
			Save:      true,
		})
		checksum := sha256.Sum256(batchBuffer)
		checksums = append(checksums, checksum[:])
	}

	ch.batchHeader.End = blockHeight
	ch.batchHeader.Chunks = checksums
	headerBytes, err := json.Marshal(ch.batchHeader)
	if err != nil {
		return err
	}

	msg, err := ch.createBatchMsg(headerBytes)
	if err != nil {
		return err
	}
	ch.processedMsgs[0].Msgs = []sdk.Msg{msg}
	return nil
}

func (ch *Child) createBatchMsg(batchBytes []byte) (sdk.Msg, error) {
	submitter, err := ch.da.GetAddressStr()
	if err != nil {
		return nil, err
	}

	return ophosttypes.NewMsgRecordBatch(
		submitter,
		ch.bridgeInfo.BridgeId,
		batchBytes,
	), nil
}
