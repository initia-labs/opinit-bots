package eventhandler

import (
	"maps"
	"sort"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
)

func (ch *ChallengeEventHandler) GetPendingEvent(id challengertypes.ChallengeId) (challengertypes.ChallengeEvent, bool) {
	ch.pendingEventsMu.Lock()
	defer ch.pendingEventsMu.Unlock()

	event, ok := ch.pendingEvents[id]
	return event, ok
}

func (ch *ChallengeEventHandler) DeletePendingEvents(events []challengertypes.ChallengeEvent) {
	ch.pendingEventsMu.Lock()
	defer ch.pendingEventsMu.Unlock()

	for _, event := range events {
		delete(ch.pendingEvents, event.Id())
	}
}

func (ch *ChallengeEventHandler) DeletePendingEvent(id challengertypes.ChallengeId) {
	ch.pendingEventsMu.Lock()
	defer ch.pendingEventsMu.Unlock()

	delete(ch.pendingEvents, id)
}

func (ch *ChallengeEventHandler) getOraclePendingEvents(l1BlockHeight uint64) []challengertypes.ChallengeEvent {
	ch.pendingEventsMu.Lock()
	defer ch.pendingEventsMu.Unlock()

	events := make([]challengertypes.ChallengeEvent, 0)
	for _, event := range ch.pendingEvents {
		if event.Type() == challengertypes.EventTypeOracle && event.Id().Id < l1BlockHeight {
			events = append(events, event)
		}
	}
	return events
}

func (ch *ChallengeEventHandler) NumPendingEvents() map[string]int64 {
	ch.pendingEventsMu.Lock()
	defer ch.pendingEventsMu.Unlock()

	numPendingEvents := make(map[string]int64)
	for _, event := range ch.pendingEvents {
		numPendingEvents[event.Type().String()]++
	}
	return numPendingEvents
}

func (ch *ChallengeEventHandler) GetAllPendingEvents() []challengertypes.ChallengeEvent {
	return ch.GetUnprocessedPendingEvents(nil)
}

func (ch *ChallengeEventHandler) GetAllSortedPendingEvents() []challengertypes.ChallengeEvent {
	pendingEvents := ch.GetAllPendingEvents()
	sort.Slice(pendingEvents, func(i, j int) bool {
		if pendingEvents[i].Type() == pendingEvents[j].Type() {
			return pendingEvents[i].Id().Id < pendingEvents[j].Id().Id
		}
		return pendingEvents[i].Type() < pendingEvents[j].Type()
	})
	return pendingEvents
}

func (ch *ChallengeEventHandler) GetUnprocessedPendingEvents(processedEvents []challengertypes.ChallengeEvent) []challengertypes.ChallengeEvent {
	ch.pendingEventsMu.Lock()
	defer ch.pendingEventsMu.Unlock()

	copiedPendingEvents := maps.Clone(ch.pendingEvents)
	for _, event := range processedEvents {
		delete(copiedPendingEvents, event.Id())
	}

	unprocessedPendingEvents := make([]challengertypes.ChallengeEvent, 0, len(copiedPendingEvents))
	for _, event := range copiedPendingEvents {
		unprocessedPendingEvents = append(unprocessedPendingEvents, event)
	}
	return unprocessedPendingEvents
}

func (ch *ChallengeEventHandler) SetPendingEvents(events []challengertypes.ChallengeEvent) {
	ch.pendingEventsMu.Lock()
	defer ch.pendingEventsMu.Unlock()

	for _, event := range events {
		ch.pendingEvents[event.Id()] = event
	}
}

func (ch *ChallengeEventHandler) loadPendingEvents() (events []challengertypes.ChallengeEvent, err error) {
	iterErr := ch.db.PrefixedIterate(challengertypes.PendingEventKey, nil, func(key, value []byte) (stop bool, err error) {
		id, err := challengertypes.ParsePendingEvent(key)
		if err != nil {
			return true, err
		}

		event, err := challengertypes.UnmarshalChallengeEvent(id.Type, value)
		if err != nil {
			return true, err
		}
		events = append(events, event)
		return false, nil
	})
	if iterErr != nil {
		return nil, iterErr
	}
	return
}
