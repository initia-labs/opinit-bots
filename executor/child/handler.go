package child

import (
	"time"

	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
	"github.com/initia-labs/opinit-bots-go/types"
)

func (ch *Child) beginBlockHandler(args nodetypes.BeginBlockArgs) (err error) {
	// just to make sure that childMsgQueue is empty
	if args.BlockHeight == args.LatestHeight && len(ch.msgQueue) != 0 && len(ch.processedMsgs) != 0 {
		panic("must not happen, msgQueue should be empty")
	}

	err = ch.prepareWithdrawals(args.BlockHeight)
	if err != nil {
		return err
	}
	return nil
}

func (ch *Child) endBlockHandler(args nodetypes.EndBlockArgs) error {
	// temporary 50 limit for msg queue
	// collect more msgs if block height is not latest
	if args.BlockHeight != args.LatestHeight && len(ch.msgQueue) <= 50 {
		return nil
	}

	batchKVs := []types.KV{
		ch.node.RawKVSyncInfo(args.BlockHeight),
	}

	withdrawalsKVs, err := ch.handleBlockWithdrawals()
	if err != nil {
		return err
	}
	batchKVs = append(batchKVs, withdrawalsKVs...)

	outputkvs, err := ch.generateOutputRoot(args.BlockHeight)
	if err != nil {
		return err
	}
	batchKVs = append(batchKVs, outputkvs...)

	if ch.node.HasKey() {
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
	ch.blockWithdrawals = ch.blockWithdrawals[:0]
	return nil
}
