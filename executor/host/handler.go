package host

import (
	"time"

	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
	"github.com/initia-labs/opinit-bots-go/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (h *Host) beginBlockHandler(args nodetypes.BeginBlockArgs) error {
	blockHeight := uint64(args.BlockHeader.Height)
	// just to make sure that childMsgQueue is empty
	if blockHeight == args.LatestHeight && len(h.msgQueue) != 0 && len(h.processedMsgs) != 0 {
		panic("must not happen, msgQueue should be empty")
	}
	return nil
}

func (h *Host) endBlockHandler(args nodetypes.EndBlockArgs) error {
	// temporary 50 limit for msg queue
	// collect more msgs if block height is not latest
	blockHeight := uint64(args.BlockHeader.Height)
	if blockHeight != args.LatestHeight && len(h.msgQueue) > 0 && len(h.msgQueue) <= 10 {
		return nil
	}

	batchKVs := []types.KV{
		h.node.RawKVSyncInfo(blockHeight),
	}
	if h.node.HasKey() {
		if len(h.msgQueue) != 0 {
			h.processedMsgs = append(h.processedMsgs, nodetypes.ProcessedMsgs{
				Msgs:      h.msgQueue,
				Timestamp: time.Now().UnixNano(),
				Save:      true,
			})
		}

		msgkvs, err := h.child.RawKVProcessedData(h.processedMsgs, false)
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

func (h *Host) txHandler(args nodetypes.TxHandlerArgs) error {
	if args.BlockHeight == args.LatestHeight && args.TxIndex == 0 {
		msg, err := h.oracleTxHandler(args.BlockHeight, args.Tx)
		if err != nil {
			return err
		}

		h.processedMsgs = append(h.processedMsgs, nodetypes.ProcessedMsgs{
			Msgs:      []sdk.Msg{msg},
			Timestamp: time.Now().UnixNano(),
			Save:      false,
		})
	}
	return nil
}
