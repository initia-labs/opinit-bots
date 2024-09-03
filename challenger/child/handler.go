package child

import (
	"context"

	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/txutils"
	"github.com/initia-labs/opinit-bots/types"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
)

func (ch *Child) beginBlockHandler(ctx context.Context, args nodetypes.BeginBlockArgs) (err error) {
	blockHeight := uint64(args.Block.Header.Height)
	// just to make sure that childMsgQueue is empty
	if len(ch.eventQueue) != 0 {
		panic("must not happen, eventQueue should be empty")
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
	treeKVs, storageRoot, err := ch.handleTree(blockHeight, args.Block.Header)
	if err != nil {
		return err
	}

	batchKVs = append(batchKVs, treeKVs...)
	if storageRoot != nil {
		err = ch.handleOutput(args.Block.Header.Time, blockHeight, ch.Version(), args.BlockID, ch.Merkle().GetWorkingTreeIndex(), storageRoot)
		if err != nil {
			return err
		}
	}

	// update the sync info
	batchKVs = append(batchKVs, ch.Node().SyncInfoToRawKV(blockHeight))

	challenges, eventKVs, err := ch.checkPendingEvents(ch.eventQueue)
	if err != nil {
		return err
	}
	batchKVs = append(batchKVs, eventKVs...)

	err = ch.DB().RawBatchSet(batchKVs...)
	if err != nil {
		return err
	}

	for _, event := range ch.eventQueue {
		ch.deletePendingEvent(event.Id())
	}
	ch.handleChallenges(challenges)
	ch.eventQueue = ch.eventQueue[:0]
	return nil
}

func (ch *Child) txHandler(_ context.Context, args nodetypes.TxHandlerArgs) error {
	txConfig := ch.Node().GetTxConfig()

	tx, err := txutils.DecodeTx(txConfig, args.Tx)
	if err != nil {
		// if tx is not oracle tx, tx parse error is expected
		// ignore decoding error
		return nil
	}
	msgs := tx.GetMsgs()
	if len(msgs) > 1 {
		// we only expect one message for oracle tx
		return nil
	}
	msg, ok := msgs[0].(*opchildtypes.MsgUpdateOracle)
	if !ok {
		return nil
	}
	ch.oracleTxHandler(args.BlockTime, msg.Height, msg.Data)
	return nil
}
