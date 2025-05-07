package host

import (
	"github.com/initia-labs/opinit-bots/node"
	"github.com/initia-labs/opinit-bots/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/opinit-bots/node/broadcaster"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"

	"github.com/pkg/errors"
)

func (h *Host) beginBlockHandler(_ types.Context, _ nodetypes.BeginBlockArgs) error {
	h.EmptyMsgQueue()
	h.EmptyProcessedMsgs()
	h.stage.Reset()
	return nil
}

func (h *Host) endBlockHandler(_ types.Context, args nodetypes.EndBlockArgs) error {
	err := node.SetSyncedHeight(h.stage, args.Block.Header.Height)
	if err != nil {
		return errors.Wrap(err, "failed to set synced height")
	}

	if h.child.HasBroadcaster() {
		h.AppendProcessedMsgs(broadcaster.MsgsToProcessedMsgs(h.GetMsgQueue())...)

		// save processed msgs to stage using child db
		err := broadcaster.SaveProcessedMsgsBatch(h.stage.WithPrefixedKey(h.child.DB().PrefixedKey), h.child.Codec(), h.GetProcessedMsgs())
		if err != nil {
			return errors.Wrap(err, "failed to save processed msgs on child db")
		}
	} else {
		h.EmptyProcessedMsgs()
	}

	err = h.SaveInternalStatus(h.stage)
	if err != nil {
		return errors.Wrap(err, "failed to save internal status")
	}
	err = h.stage.Commit()
	if err != nil {
		return errors.Wrap(err, "failed to commit stage")
	}
	h.child.BroadcastProcessedMsgs(h.GetProcessedMsgs()...)
	return nil
}

func (h *Host) txHandler(_ types.Context, args nodetypes.TxHandlerArgs) error {
	if args.BlockHeight == args.LatestHeight && args.TxIndex == 0 {
		msg, sender, err := h.oracleTxHandler(args.BlockHeight, args.Tx)
		if err != nil {
			return errors.Wrap(err, "failed to handle oracle tx")
		} else if msg != nil {
			h.AppendProcessedMsgs(btypes.ProcessedMsgs{
				Sender:    sender,
				Msgs:      []sdk.Msg{msg},
				Timestamp: types.CurrentNanoTimestamp(),
				Save:      false,
			})
		}
	}
	return nil
}
