package host

import (
	"context"
	"slices"
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
	blockHeight := args.Block.Header.Height
	msgQueue := h.GetMsgQueue()

	batchKVs := []types.RawKV{
		h.Node().SyncInfoToRawKV(blockHeight),
	}
	if h.child.HasKey() {
		for i := 0; i < len(msgQueue); i += 5 {
			end := i + 5
			if end > len(msgQueue) {
				end = len(msgQueue)
			}

			h.AppendProcessedMsgs(btypes.ProcessedMsgs{
				Msgs:      slices.Clone(msgQueue[i:end]),
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
