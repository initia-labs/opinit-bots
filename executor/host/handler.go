package host

import (
	"context"
	"time"

	"github.com/initia-labs/opinit-bots/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
)

func (h *Host) beginBlockHandler(_ context.Context, args nodetypes.BeginBlockArgs) error {
	h.EmptyMsgQueue()
	h.EmptyProcessedMsgs()
	return nil
}

func (h *Host) endBlockHandler(_ context.Context, args nodetypes.EndBlockArgs) error {
	// collect more msgs if block height is not latest
	blockHeight := uint64(args.Block.Header.Height)
	msgQueue := h.GetMsgQueue()
	if blockHeight != args.LatestHeight && len(msgQueue) > 0 && len(msgQueue) <= 10 {
		return nil
	}

	batchKVs := []types.RawKV{
		h.Node().SyncInfoToRawKV(blockHeight),
	}
	if h.Node().HasBroadcaster() {
		if len(msgQueue) != 0 {
			h.AppendProcessedMsgs(btypes.ProcessedMsgs{
				Msgs:      msgQueue,
				Timestamp: time.Now().UnixNano(),
				Save:      true,
			})
		}

		msgkvs, err := h.child.ProcessedMsgsToRawKV(h.GetProcessedMsgs(), false)
		if err != nil {
			return err
		}
		batchKVs = append(batchKVs, msgkvs...)
	}

	err := h.DB().RawBatchSet(batchKVs...)
	if err != nil {
		return err
	}

	for _, processedMsg := range h.GetProcessedMsgs() {
		h.child.BroadcastMsgs(processedMsg)
	}
	return nil
}

func (h *Host) txHandler(_ context.Context, args nodetypes.TxHandlerArgs) error {
	if args.BlockHeight == args.LatestHeight && args.TxIndex == 0 {
		if msg, err := h.oracleTxHandler(args.BlockHeight, args.Tx); err != nil {
			return err
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
