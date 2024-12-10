package child

import (
	eventhandler "github.com/initia-labs/opinit-bots/challenger/eventhandler"
	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	"github.com/initia-labs/opinit-bots/node"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/txutils"
	"github.com/initia-labs/opinit-bots/types"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	challengerdb "github.com/initia-labs/opinit-bots/challenger/db"
	"github.com/pkg/errors"
)

func (ch *Child) beginBlockHandler(ctx types.Context, args nodetypes.BeginBlockArgs) (err error) {
	blockHeight := args.Block.Header.Height
	ch.eventQueue = ch.eventQueue[:0]
	ch.stage.Reset()

	err = ch.prepareTree(blockHeight)
	if err != nil {
		return errors.Wrap(err, "failed to prepare tree")
	}

	err = ch.prepareOutput(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to prepare output")
	}
	return nil
}

func (ch *Child) endBlockHandler(ctx types.Context, args nodetypes.EndBlockArgs) error {
	blockHeight := args.Block.Header.Height
	pendingChallenges := make([]challengertypes.Challenge, 0)

	storageRoot, err := ch.handleTree(ctx, blockHeight, args.Block.Header)
	if err != nil {
		return errors.Wrap(err, "failed to handle tree")
	}

	if storageRoot != nil {
		workingTree, err := ch.WorkingTree()
		if err != nil {
			return errors.Wrap(err, "failed to get working tree")
		}

		err = ch.handleOutput(args.Block.Header.Time, blockHeight, ch.Version(), args.BlockID, workingTree.Index, storageRoot)
		if err != nil {
			return errors.Wrap(err, "failed to handle output")
		}
	}

	// update the sync info
	err = node.SetSyncedHeight(ch.stage, args.Block.Header.Height)
	if err != nil {
		return errors.Wrap(err, "failed to set synced height")
	}

	// check value for pending events
	challenges, processedEvents, err := ch.eventHandler.CheckValue(ctx, ch.eventQueue)
	if err != nil {
		return err
	}
	pendingChallenges = append(pendingChallenges, challenges...)

	// check timeout for unprocessed pending events
	unprocessedEvents := ch.eventHandler.GetUnprocessedPendingEvents(processedEvents)
	challenges, timeoutEvents := ch.eventHandler.CheckTimeout(args.Block.Header.Time, unprocessedEvents)
	pendingChallenges = append(pendingChallenges, challenges...)

	err = eventhandler.SavePendingEvents(ch.stage, timeoutEvents)
	if err != nil {
		return err
	}

	// delete processed events
	err = eventhandler.DeletePendingEvents(ch.stage, processedEvents)
	if err != nil {
		return err
	}

	err = challengerdb.SavePendingChallenges(ch.stage.WithPrefixedKey(ch.challenger.DB().PrefixedKey), pendingChallenges)
	if err != nil {
		return errors.Wrap(err, "failed to save pending events on child db")
	}

	err = ch.stage.Commit()
	if err != nil {
		return err
	}

	ch.eventHandler.DeletePendingEvents(processedEvents)
	ch.eventHandler.SetPendingEvents(timeoutEvents)
	ch.challenger.SendPendingChallenges(challenges)
	return nil
}

func (ch *Child) txHandler(ctx types.Context, args nodetypes.TxHandlerArgs) error {
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
	ch.oracleTxHandler(ctx, args.BlockTime, msg.Sender, types.MustUint64ToInt64(msg.Height), msg.Data)
	return nil
}
