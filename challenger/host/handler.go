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

	prevEvents := make([]challengertypes.ChallengeEvent, 0)

	if len(h.outputPendingEventQueue) > 0 {
		// save the last output event to host db
		err = eventhandler.SavePendingEvent(h.stage, h.outputPendingEventQueue[len(h.outputPendingEventQueue)-1])
		if err != nil {
			return errors.Wrap(err, "failed to save pending event on host db")
		}

		prevEvent, ok := h.eventHandler.GetPrevPendingEvent(h.outputPendingEventQueue[0])
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

	err = h.SaveInternalStatus(h.stage)
	if err != nil {
		return errors.Wrap(err, "failed to save internal status")
	}

	err = h.stage.Commit()
	if err != nil {
		return errors.Wrap(err, "failed to commit stage")
	}

	h.child.SetPendingEvents(h.eventQueue)
	h.eventHandler.DeletePendingEvents(processedEvents)
	// save the last output event
	if len(h.outputPendingEventQueue) > 0 {
		h.eventHandler.SetPendingEvent(h.outputPendingEventQueue[len(h.outputPendingEventQueue)-1])
	}
	h.challenger.SendPendingChallenges(pendingChallenges)
	return nil
}

func (h *Host) txHandler(_ types.Context, args nodetypes.TxHandlerArgs) error {
	if args.TxIndex == 0 {
		h.oracleTxHandler(args.BlockHeight, args.BlockTime, args.Tx)
	}
	return nil
}
