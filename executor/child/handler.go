package child

import (
	"context"
	"time"

	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"
)

func (ch *Child) beginBlockHandler(ctx context.Context, args nodetypes.BeginBlockArgs) (err error) {
	blockHeight := uint64(args.Block.Header.Height)
	// just to make sure that childMsgQueue is empty
	if blockHeight == args.LatestHeight && len(ch.GetMsgQueue()) != 0 && len(ch.GetProcessedMsgs()) != 0 {
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
		err = ch.handleOutput(blockHeight, ch.Version(), args.BlockID, ch.Merkle().GetWorkingTreeIndex(), storageRoot)
		if err != nil {
			return err
		}
	}

	// if we are in sync and we have a small number of messages, less than 10,
	// then store the current updates in the database and process the next block.
	if blockHeight < args.LatestHeight && len(ch.GetMsgQueue()) > 0 && len(ch.GetMsgQueue()) <= 10 {
		return ch.DB().RawBatchSet(batchKVs...)
	}

	// update the sync info
	batchKVs = append(batchKVs, ch.Node().SyncInfoToRawKV(blockHeight))

	// if has key, then process the messages
	if ch.host.HasKey() {
		if len(ch.GetMsgQueue()) != 0 {
			ch.AppendProcessedMsgs(btypes.ProcessedMsgs{
				Msgs:      ch.GetMsgQueue(),
				Timestamp: time.Now().UnixNano(),
				Save:      true,
			})
		}
		msgKVs, err := ch.host.ProcessedMsgsToRawKV(ch.GetProcessedMsgs(), false)
		if err != nil {
			return err
		}
		batchKVs = append(batchKVs, msgKVs...)
	}

	err = ch.DB().RawBatchSet(batchKVs...)
	if err != nil {
		return err
	}

	for _, processedMsg := range ch.GetProcessedMsgs() {
		ch.host.BroadcastMsgs(processedMsg)
	}

	ch.EmptyMsgQueue()
	ch.EmptyProcessedMsgs()
	return nil
}
