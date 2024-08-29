package host

import (
	"context"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"
)

func (h *Host) beginBlockHandler(_ context.Context, args nodetypes.BeginBlockArgs) error {
	if len(h.elemQueue) != 0 {
		panic("must not happen, eventQueue should be empty")
	}
	return nil
}

func (h *Host) endBlockHandler(_ context.Context, args nodetypes.EndBlockArgs) error {
	blockHeight := uint64(args.Block.Header.Height)
	batchKVs := []types.RawKV{
		h.Node().SyncInfoToRawKV(blockHeight),
	}
	for _, elem := range h.elemQueue {
		value, err := elem.Event.Marshal()
		if err != nil {
			return err
		}
		batchKVs = append(batchKVs, types.RawKV{
			Key:   h.DB().PrefixedKey(challengertypes.PrefixedChallengeElem(elem)),
			Value: value,
		})
	}

	err := h.DB().RawBatchSet(batchKVs...)
	if err != nil {
		return err
	}

	for _, elem := range h.elemQueue {
		h.elemCh <- elem
	}
	h.elemQueue = h.elemQueue[:0]
	return nil
}
