package host

import (
	"time"

	"github.com/initia-labs/opinit-bots/node"
	"github.com/initia-labs/opinit-bots/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/opinit-bots/node/broadcaster"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"

	"github.com/pkg/errors"
)

func (h *Host) beginBlockHandler(_ types.Context, args nodetypes.BeginBlockArgs) error {
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
		err := h.stage.ExecuteFnWithDB(h.child.DB(), func() error {
			return broadcaster.SaveProcessedMsgsBatch(h.stage, h.child.Codec(), h.GetProcessedMsgs())
		})
		if err != nil {
			return errors.Wrap(err, "failed to save processed msgs on child db")
		}
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
		if msg, err := h.oracleTxHandler(args.BlockHeight, args.Tx); err != nil {
			return errors.Wrap(err, "failed to handle oracle tx")
		} else if msg != nil {
			h.AppendProcessedMsgs(btypes.ProcessedMsgs{
				Msgs:      []sdk.Msg{msg},
				Timestamp: time.Now().UnixNano(),
				Save:      false,
			})
		}
	}
	return nil
}
