package host

import (
	"context"
	"time"

	"github.com/initia-labs/opinit-bots-go/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	btypes "github.com/initia-labs/opinit-bots-go/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
)

func (h *Host) beginBlockHandler(_ context.Context, args nodetypes.BeginBlockArgs) error {
	blockHeight := uint64(args.Block.Header.Height)
	// just to make sure that childMsgQueue is empty
	if blockHeight == args.LatestHeight && len(h.msgQueue) != 0 && len(h.processedMsgs) != 0 {
		panic("must not happen, msgQueue should be empty")
	}
	return nil
}

func (h *Host) endBlockHandler(_ context.Context, args nodetypes.EndBlockArgs) error {
	// temporary 50 limit for msg queue
	// collect more msgs if block height is not latest
	blockHeight := uint64(args.Block.Header.Height)
	if blockHeight != args.LatestHeight && len(h.msgQueue) > 0 && len(h.msgQueue) <= 10 {
		return nil
	}

	batchKVs := []types.RawKV{
		h.node.SyncInfoToRawKV(blockHeight),
	}
	if h.node.HasBroadcaster() {
		if len(h.msgQueue) != 0 {
			h.processedMsgs = append(h.processedMsgs, btypes.ProcessedMsgs{
				Msgs:      h.msgQueue,
				Timestamp: time.Now().UnixNano(),
				Save:      true,
			})
		}

		msgkvs, err := h.child.ProcessedMsgsToRawKV(h.processedMsgs, false)
		if err != nil {
			return err
		}
		batchKVs = append(batchKVs, msgkvs...)
	}

	err := h.db.RawBatchSet(batchKVs...)
	if err != nil {
		return err
	}

	for _, processedMsg := range h.processedMsgs {
		h.child.BroadcastMsgs(processedMsg)
	}

	h.msgQueue = h.msgQueue[:0]
	h.processedMsgs = h.processedMsgs[:0]
	return nil
}

func (h *Host) txHandler(_ context.Context, args nodetypes.TxHandlerArgs) error {
	if args.BlockHeight == args.LatestHeight && args.TxIndex == 0 {
		if msg, err := h.oracleTxHandler(args.BlockHeight, args.Tx); err != nil {
			return err
		} else if msg != nil {
			h.processedMsgs = append(h.processedMsgs, btypes.ProcessedMsgs{
				Msgs:      []sdk.Msg{msg},
				Timestamp: time.Now().UnixNano(),
				Save:      false,
			})
		}
	}
	return nil
}
