package child

import (
	"time"

	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
	"github.com/initia-labs/opinit-bots-go/types"
)

func (ch *Child) beginBlockHandler(args nodetypes.BeginBlockArgs) (err error) {
	blockHeight := uint64(args.Block.Header.Height)
	// just to make sure that childMsgQueue is empty
	if blockHeight == args.LatestHeight && len(ch.msgQueue) != 0 && len(ch.processedMsgs) != 0 {
		panic("must not happen, msgQueue should be empty")
	}

	err = ch.prepareTree(blockHeight)
	if err != nil {
		return err
	}
	err = ch.prepareOutput()
	if err != nil {
		return err
	}
	return nil
}

func (ch *Child) endBlockHandler(args nodetypes.EndBlockArgs) error {
	blockHeight := uint64(args.Block.Header.Height)
	batchKVs := make([]types.KV, 0)
	treeKVs, storageRoot, err := ch.handleTree(blockHeight, uint64(args.LatestHeight), args.BlockID, args.Block.Header)
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
	// collect more msgs if block height is not latest
	if blockHeight != args.LatestHeight && len(ch.msgQueue) > 0 && len(ch.msgQueue) <= 10 {
		return ch.db.RawBatchSet(batchKVs...)
	}

	batchKVs = append(batchKVs, ch.node.RawKVSyncInfo(blockHeight))

	if ch.host.HasKey() {
		if len(ch.msgQueue) != 0 {
			ch.processedMsgs = append(ch.processedMsgs, nodetypes.ProcessedMsgs{
				Msgs:      ch.msgQueue,
				Timestamp: time.Now().UnixNano(),
				Save:      true,
			})
		}
		msgkvs, err := ch.host.RawKVProcessedData(ch.processedMsgs, false)
		if err != nil {
			return err
		}
		batchKVs = append(batchKVs, msgkvs...)
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
