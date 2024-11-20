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
	err = h.stage.ExecuteFnWithDB(h.child.DB(), func() error {
		return eventhandler.SavePendingEvents(h.stage, h.eventQueue)
	})
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
	pendingChallenges, precessedEvents := h.eventHandler.CheckTimeout(args.Block.Header.Time, unprocessedEvents)
	precessedEvents = append(precessedEvents, prevEvents...)

	// delete processed events
	err = eventhandler.DeletePendingEvents(h.stage, precessedEvents)
	if err != nil {
		return err
	}

	err = h.stage.ExecuteFnWithDB(h.challenger.DB(), func() error {
		return challengerdb.SavePendingChallenges(h.stage, pendingChallenges)
	})
	if err != nil {
		return errors.Wrap(err, "failed to save pending events on child db")
	}

	err = h.stage.Commit()
	if err != nil {
		return errors.Wrap(err, "failed to commit stage")
	}

	h.child.SetPendingEvents(h.eventQueue)
	h.eventHandler.DeletePendingEvents(precessedEvents)
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
