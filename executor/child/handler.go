package child

import (
	"time"

	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
	"github.com/initia-labs/opinit-bots-go/types"
)

func (ch *Child) beginBlockHandler(args nodetypes.BeginBlockArgs) (err error) {
	blockHeight := uint64(args.BlockHeader.Height)
	// just to make sure that childMsgQueue is empty
	if blockHeight == args.LatestHeight && len(ch.msgQueue) != 0 && len(ch.processedMsgs) != 0 {
		panic("must not happen, msgQueue should be empty")
	}

	err = ch.prepareWithdrawals(blockHeight, args.BlockHeader.Time)
	if err != nil {
		return err
	}
	return nil
}

func (ch *Child) endBlockHandler(args nodetypes.EndBlockArgs) error {
	err := ch.mk.SaveWorkingTree(uint64(args.BlockHeader.Height))
	if err != nil {
		return err
	}

	// temporary 50 limit for msg queue
	// collect more msgs if block height is not latest
	blockHeight := uint64(args.BlockHeader.Height)
	if blockHeight != args.LatestHeight && len(ch.msgQueue) <= 50 {
		return nil
	}

	batchKVs := []types.KV{
		ch.node.RawKVSyncInfo(blockHeight),
	}

	if blockHeight == args.LatestHeight {
		outputkvs, err := ch.proposeOutput(ch.version, args.BlockID, args.BlockHeader)
		if err != nil {
			return err
		}
		batchKVs = append(batchKVs, outputkvs...)
	}

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
