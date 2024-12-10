package host

import (
	challengerdb "github.com/initia-labs/opinit-bots/challenger/db"
	"github.com/initia-labs/opinit-bots/challenger/eventhandler"
	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	"github.com/initia-labs/opinit-bots/node"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"
)

func (h *Host) beginBlockHandler(_ types.Context, args nodetypes.BeginBlockArgs) error {
	h.eventQueue = h.eventQueue[:0]
	h.outputPendingEventQueue = h.outputPendingEventQueue[:0]
	h.stage.Reset()
	return nil
}

func (h *Host) endBlockHandler(_ types.Context, args nodetypes.EndBlockArgs) error {
	err := node.SetSyncedHeight(h.stage, args.Block.Header.Height)
	if err != nil {
		return errors.Wrap(err, "failed to set synced height")
	}

	// save all pending events to child db
	err = eventhandler.SavePendingEvents(h.stage.WithPrefixedKey(h.child.DB().PrefixedKey), h.eventQueue)
	if err != nil {
		return errors.Wrap(err, "failed to save pending events on child db")
	}

	// save all pending events to host db
	// currently, only output event is considered as pending event
	if len(h.outputPendingEventQueue) > 1 || (len(h.outputPendingEventQueue) == 1 && h.outputPendingEventQueue[0].Type() != challengertypes.EventTypeOutput) {
		panic("must not happen, outputPendingEventQueue should have only one output event")
	}

	err = eventhandler.SavePendingEvents(h.stage, h.outputPendingEventQueue)
	if err != nil {
		return err
	}

	prevEvents := make([]challengertypes.ChallengeEvent, 0)
	for _, pendingEvent := range h.outputPendingEventQueue {
		prevEvent, ok := h.eventHandler.GetPrevPendingEvent(pendingEvent)
		if ok {
			prevEvents = append(prevEvents, prevEvent)
		}
	}
	unprocessedEvents := h.eventHandler.GetUnprocessedPendingEvents(prevEvents)
	pendingChallenges, processedEvents := h.eventHandler.CheckTimeout(args.Block.Header.Time, unprocessedEvents)
	processedEvents = append(processedEvents, prevEvents...)

	// delete processed events
	err = eventhandler.DeletePendingEvents(h.stage, processedEvents)
	if err != nil {
		return err
	}

	err = challengerdb.SavePendingChallenges(h.stage.WithPrefixedKey(h.challenger.DB().PrefixedKey), pendingChallenges)
	if err != nil {
		return errors.Wrap(err, "failed to save pending events on child db")
	}

	err = h.stage.Commit()
	if err != nil {
		return errors.Wrap(err, "failed to commit stage")
	}

	h.child.SetPendingEvents(h.eventQueue)
	h.eventHandler.DeletePendingEvents(processedEvents)
	h.eventHandler.SetPendingEvents(h.outputPendingEventQueue)
	h.challenger.SendPendingChallenges(pendingChallenges)
	return nil
}

func (h *Host) txHandler(_ types.Context, args nodetypes.TxHandlerArgs) error {
	if args.TxIndex == 0 {
		h.oracleTxHandler(args.BlockHeight, args.BlockTime, args.Tx)
	}
	return nil
}
