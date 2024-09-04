package host

import (
	"context"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"
)

func (h *Host) beginBlockHandler(_ context.Context, args nodetypes.BeginBlockArgs) error {
	h.eventQueue = h.eventQueue[:0]
	h.outputPendingEventQueue = h.outputPendingEventQueue[:0]
	return nil
}

func (h *Host) endBlockHandler(_ context.Context, args nodetypes.EndBlockArgs) error {
	blockHeight := uint64(args.Block.Header.Height)
	batchKVs := []types.RawKV{
		h.Node().SyncInfoToRawKV(blockHeight),
	}

	// save all pending events to child db
	eventKVs, err := h.child.PendingEventsToRawKV(h.eventQueue, false)
	if err != nil {
		return err
	}
	batchKVs = append(batchKVs, eventKVs...)

	// save all pending events to host db
	// currently, only output event is considered as pending event
	if len(h.outputPendingEventQueue) > 1 || (len(h.outputPendingEventQueue) == 1 && h.outputPendingEventQueue[0].Type() != challengertypes.EventTypeOutput) {
		panic("must not happen, outputPendingEventQueue should have only one output event")
	}

	eventKVs, err = h.eventHandler.PendingEventsToRawKV(h.outputPendingEventQueue, false)
	if err != nil {
		return err
	}
	batchKVs = append(batchKVs, eventKVs...)

	prevEvents := make([]challengertypes.ChallengeEvent, 0)
	for _, pendingEvent := range h.outputPendingEventQueue {
		prevEvent, ok := h.eventHandler.GetPrevPendingEvent(pendingEvent)
		if ok {
			prevEvents = append(prevEvents, prevEvent)
		}
	}
	unprocessedEvents := h.eventHandler.GetUnprocessedPendingEvents(prevEvents)
	pendingChallenges, precessedEvents := h.eventHandler.CheckTimeout(args.Block.Header.Time, unprocessedEvents)
	precessedEvents = append(precessedEvents, prevEvents...)

	// delete processed events
	eventKVs, err = h.eventHandler.PendingEventsToRawKV(precessedEvents, true)
	if err != nil {
		return err
	}
	batchKVs = append(batchKVs, eventKVs...)

	challengesKVs, err := h.challenger.PendingChallengeToRawKVs(pendingChallenges, false)
	if err != nil {
		return err
	}
	batchKVs = append(batchKVs, challengesKVs...)

	err = h.DB().RawBatchSet(batchKVs...)
	if err != nil {
		return err
	}

	h.child.SetPendingEvents(h.eventQueue)
	h.eventHandler.DeletePendingEvents(precessedEvents)
	h.eventHandler.SetPendingEvents(h.outputPendingEventQueue)
	h.challenger.SendPendingChallenges(pendingChallenges)
	return nil
}

func (h *Host) txHandler(_ context.Context, args nodetypes.TxHandlerArgs) error {
	if args.TxIndex == 0 {
		h.oracleTxHandler(args.BlockHeight, args.BlockTime, args.Tx)
	}
	return nil
}
