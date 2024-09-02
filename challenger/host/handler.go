package host

import (
	"context"

	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"
)

func (h *Host) beginBlockHandler(_ context.Context, args nodetypes.BeginBlockArgs) error {
	if len(h.eventQueue) != 0 {
		panic("must not happen, eventQueue should be empty")
	}
	return nil
}

func (h *Host) endBlockHandler(_ context.Context, args nodetypes.EndBlockArgs) error {
	blockHeight := uint64(args.Block.Header.Height)
	batchKVs := []types.RawKV{
		h.Node().SyncInfoToRawKV(blockHeight),
	}

	eventKVs, err := h.child.PendingEventsToRawKV(h.eventQueue, false)
	if err != nil {
		return err
	}
	batchKVs = append(batchKVs, eventKVs...)

	err = h.DB().RawBatchSet(batchKVs...)
	if err != nil {
		return err
	}

	h.child.SetPendingEvents(h.eventQueue)
	h.eventQueue = h.eventQueue[:0]
	return nil
}

func (h *Host) txHandler(_ context.Context, args nodetypes.TxHandlerArgs) error {
	if args.TxIndex == 0 {
		h.oracleTxHandler(args.BlockHeight, args.BlockTime, args.Tx)
	}
	return nil
}
