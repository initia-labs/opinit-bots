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

	authz "github.com/cosmos/cosmos-sdk/x/authz"
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
	ch.challenger.SendPendingChallenges(pendingChallenges)
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

	// iterate through all msgs to find oracle-related ones
	// oracle relay tx may contain multiple msgs (e.g., MsgUpdateClient + MsgExec wrapping MsgRelayOracleData)
	for _, m := range msgs {
		switch msg := m.(type) {
		case *opchildtypes.MsgUpdateOracle:
			ch.oracleTxHandler(ctx, args.BlockTime, msg.Sender, types.MustUint64ToInt64(msg.Height), msg.Data)
		case *opchildtypes.MsgRelayOracleData:
			ch.oracleRelayTxHandler(ctx, args.BlockTime, msg.Sender, msg.OracleData)
		case *authz.MsgExec:
			if len(msg.Msgs) != 1 {
				continue
			}

			switch msg.Msgs[0].TypeUrl {
			case "/opinit.opchild.v1.MsgUpdateOracle":
				oracleMsg := new(opchildtypes.MsgUpdateOracle)
				err = oracleMsg.Unmarshal(msg.Msgs[0].Value)
				if err != nil {
					return err
				}
				ch.oracleTxHandler(ctx, args.BlockTime, msg.Grantee, types.MustUint64ToInt64(oracleMsg.Height), oracleMsg.Data)
			case "/opinit.opchild.v1.MsgRelayOracleData":
				relayMsg := new(opchildtypes.MsgRelayOracleData)
				err = relayMsg.Unmarshal(msg.Msgs[0].Value)
				if err != nil {
					return err
				}
				ch.oracleRelayTxHandler(ctx, args.BlockTime, msg.Grantee, relayMsg.OracleData)
			}
		}
	}

	return nil
}
