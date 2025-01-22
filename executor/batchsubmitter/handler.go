package batchsubmitter

import (
	"github.com/pkg/errors"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/cosmos/gogoproto/proto"

	"github.com/initia-labs/opinit-bots/node"
	"github.com/initia-labs/opinit-bots/node/broadcaster"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"
)

func (bs *BatchSubmitter) rawBlockHandler(ctx types.Context, args nodetypes.RawBlockArgs) error {
	// clear processed messages
	bs.processedMsgs = bs.processedMsgs[:0]
	bs.stage.Reset()

	err := bs.prepareBatch(args.BlockHeight)
	if err != nil {
		return errors.Wrap(err, "failed to prepare batch")
	}

	pbb := new(cmtproto.Block)
	err = proto.Unmarshal(args.BlockBytes, pbb)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal block")
	}

	pbb, err = bs.emptyOracleData(pbb)
	if err != nil {
		return errors.Wrap(err, "failed to empty oracle data")
	}

	pbb, err = bs.emptyUpdateClientData(ctx, pbb)
	if err != nil {
		return errors.Wrap(err, "failed to empty update client data")
	}

	// convert block to bytes
	blockBytes, err := proto.Marshal(pbb)
	if err != nil {
		return errors.Wrap(err, "failed to marshal block")
	}

	_, err = bs.handleBatch(blockBytes)
	if err != nil {
		return errors.Wrap(err, "failed to handle batch")
	}

	fileSize, err := bs.batchFileSize(true)
	if err != nil {
		return errors.Wrap(err, "failed to get batch file size")
	}
	bs.localBatchInfo.BatchSize = fileSize

	if bs.checkBatch(args.BlockHeight, args.LatestHeight, pbb.Header.Time) {
		// finalize the batch
		bs.LastBatchEndBlockNumber = args.BlockHeight
		bs.localBatchInfo.LastSubmissionTime = pbb.Header.Time
		bs.localBatchInfo.End = args.BlockHeight

		err := bs.finalizeBatch(ctx, args.BlockHeight)
		if err != nil {
			return errors.Wrap(err, "failed to finalize batch")
		}
	}

	// store the processed state into db with batch operation
	err = node.SetSyncedHeight(bs.stage, args.BlockHeight)
	if err != nil {
		return errors.Wrap(err, "failed to set synced height")
	}
	if bs.da.HasBroadcaster() {
		// save processed msgs to stage using host db
		err := broadcaster.SaveProcessedMsgsBatch(bs.stage.WithPrefixedKey(bs.da.DB().PrefixedKey), bs.da.Codec(), bs.processedMsgs)
		if err != nil {
			return errors.Wrap(err, "failed to save processed msgs")
		}
	} else {
		bs.processedMsgs = bs.processedMsgs[:0]
	}
	err = SaveLocalBatchInfo(bs.stage, *bs.localBatchInfo)
	if err != nil {
		return errors.Wrap(err, "failed to save local batch info")
	}

	err = bs.stage.Commit()
	if err != nil {
		return errors.Wrap(err, "failed to commit stage")
	}
	// broadcast processed messages
	bs.da.BroadcastProcessedMsgs(bs.processedMsgs...)
	return nil
}
