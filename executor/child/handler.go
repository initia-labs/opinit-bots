package child

import (
	"context"
	"errors"
	"slices"
	"time"

	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"golang.org/x/exp/maps"
)

func (ch *Child) beginBlockHandler(ctx context.Context, args nodetypes.BeginBlockArgs) (err error) {
	blockHeight := args.Block.Header.Height
	ch.EmptyMsgQueue()
	ch.EmptyProcessedMsgs()
	ch.batchKVs = ch.batchKVs[:0]
	maps.Clear(ch.addressIndexMap)

	if ch.Merkle() == nil {
		return errors.New("merkle is not initialized")
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
	blockHeight := args.Block.Header.Height
	treeKVs, storageRoot, err := ch.handleTree(blockHeight, args.LatestHeight, args.BlockID, args.Block.Header)
	if err != nil {
		return err
	}

	ch.batchKVs = append(ch.batchKVs, treeKVs...)
	if storageRoot != nil {
		workingTreeIndex, err := ch.GetWorkingTreeIndex()
		if err != nil {
			return err
		}
		err = ch.handleOutput(blockHeight, ch.Version(), args.BlockID, workingTreeIndex, storageRoot)
		if err != nil {
			return err
		}
	}

	// update the sync info
	ch.batchKVs = append(ch.batchKVs, ch.Node().SyncInfoToRawKV(blockHeight))

	// if has key, then process the messages
	if ch.host.HasKey() {
		msgQueues := ch.GetMsgQueue()

		for sender := range msgQueues {
			msgQueue := msgQueues[sender]
			for i := 0; i < len(msgQueue); i += 5 {
				end := i + 5
				if end > len(msgQueue) {
					end = len(msgQueue)
				}

				ch.AppendProcessedMsgs(btypes.ProcessedMsgs{
					Msgs:      slices.Clone(msgQueue[i:end]),
					Timestamp: time.Now().UnixNano(),
					Save:      true,
				})
			}
		}

		msgKVs, err := ch.host.ProcessedMsgsToRawKV(ch.GetProcessedMsgs(), false)
		if err != nil {
			return err
		}
		ch.batchKVs = append(ch.batchKVs, msgKVs...)
	}

	err = ch.DB().RawBatchSet(ch.batchKVs...)
	if err != nil {
		return err
	}

	for _, processedMsg := range ch.GetProcessedMsgs() {
		ch.host.BroadcastMsgs(processedMsg)
	}
	return nil
}
