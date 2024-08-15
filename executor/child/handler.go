package child

import (
	"context"
	"time"

	btypes "github.com/initia-labs/opinit-bots-go/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
	"github.com/initia-labs/opinit-bots-go/types"
)

func (ch *Child) beginBlockHandler(ctx context.Context, args nodetypes.BeginBlockArgs) (err error) {
	blockHeight := uint64(args.Block.Header.Height)
	// just to make sure that childMsgQueue is empty
	if blockHeight == args.LatestHeight && len(ch.msgQueue) != 0 && len(ch.processedMsgs) != 0 {
		panic("must not happen, msgQueue should be empty")
	}

	err = ch.prepareTree(blockHeight)
	if err != nil {
		return err
	}

	err = ch.prepareOutput(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (ch *Child) endBlockHandler(_ context.Context, args nodetypes.EndBlockArgs) error {
	blockHeight := uint64(args.Block.Header.Height)
	batchKVs := make([]types.RawKV, 0)
	treeKVs, storageRoot, err := ch.handleTree(blockHeight, args.LatestHeight, args.BlockID, args.Block.Header)
	if err != nil {
		return err
	}

	batchKVs = append(batchKVs, treeKVs...)

	if storageRoot != nil {
		err = ch.handleOutput(blockHeight, ch.version, args.BlockID, ch.mk.GetWorkingTreeIndex(), storageRoot)
		if err != nil {
			return err
		}
	}

	// if we are in sync and we have a small number of messages, less than 10,
	// then store the current updates in the database and process the next block.
	if blockHeight < args.LatestHeight && len(ch.msgQueue) > 0 && len(ch.msgQueue) <= 10 {
		return ch.db.RawBatchSet(batchKVs...)
	}

	// update the sync info
	batchKVs = append(batchKVs, ch.node.SyncInfoToRawKV(blockHeight))

	// if has key, then process the messages
	if ch.host.HasKey() {
		if len(ch.msgQueue) != 0 {
			ch.processedMsgs = append(ch.processedMsgs, btypes.ProcessedMsgs{
				Msgs:      ch.msgQueue,
				Timestamp: time.Now().UnixNano(),
				Save:      true,
			})
		}
		msgKVs, err := ch.host.ProcessedMsgsToRawKV(ch.processedMsgs, false)
		if err != nil {
			return err
		}
		batchKVs = append(batchKVs, msgKVs...)
	}

	err = ch.db.RawBatchSet(batchKVs...)
	if err != nil {
		return err
	}

	for _, processedMsg := range ch.processedMsgs {
		ch.host.BroadcastMsgs(processedMsg)
	}

	ch.msgQueue = ch.msgQueue[:0]
	ch.processedMsgs = ch.processedMsgs[:0]

	return nil
}
