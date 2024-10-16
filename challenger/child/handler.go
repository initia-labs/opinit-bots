package child

import (
	"context"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/txutils"
	"github.com/initia-labs/opinit-bots/types"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
)

func (ch *Child) beginBlockHandler(ctx context.Context, args nodetypes.BeginBlockArgs) (err error) {
	blockHeight := args.Block.Header.Height
	ch.eventQueue = ch.eventQueue[:0]

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
	batchKVs := make([]types.RawKV, 0)
	pendingChallenges := make([]challengertypes.Challenge, 0)

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

	// check value for pending events
	challenges, processedEvents, err := ch.eventHandler.CheckValue(ch.eventQueue)
	if err != nil {
		return err
	}
	pendingChallenges = append(pendingChallenges, challenges...)

	// check timeout for unprocessed pending events
	unprocessedEvents := ch.eventHandler.GetUnprocessedPendingEvents(processedEvents)
	challenges, timeoutEvents := ch.eventHandler.CheckTimeout(args.Block.Header.Time, unprocessedEvents)
	pendingChallenges = append(pendingChallenges, challenges...)

	// update timeout pending events
	eventKvs, err := ch.PendingEventsToRawKV(timeoutEvents, false)
	if err != nil {
		return err
	}
	batchKVs = append(batchKVs, eventKvs...)

	// delete processed events
	eventKVs, err := ch.PendingEventsToRawKV(processedEvents, true)
	if err != nil {
		return err
	}
	batchKVs = append(batchKVs, eventKVs...)

	challengesKVs, err := ch.challenger.PendingChallengeToRawKVs(pendingChallenges, false)
	if err != nil {
		return err
	}
	batchKVs = append(batchKVs, challengesKVs...)

	err = ch.DB().RawBatchSet(batchKVs...)
	if err != nil {
		return err
	}

	ch.eventHandler.DeletePendingEvents(processedEvents)
	ch.eventHandler.SetPendingEvents(timeoutEvents)
	ch.challenger.SendPendingChallenges(challenges)
	return nil
}

func (ch *Child) txHandler(_ context.Context, args nodetypes.TxHandlerArgs) error {
	// ignore failed tx
	if !args.Success {
		return nil
	}
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
	ch.oracleTxHandler(args.BlockTime, msg.Sender, types.MustUint64ToInt64(msg.Height), msg.Data)
	return nil
}
