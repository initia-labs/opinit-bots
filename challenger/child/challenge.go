package child

import (
	"encoding/json"
	"fmt"
	"sort"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func (ch *Child) handleChallenges(challenges []challengertypes.Challenge) {
	for _, challenge := range challenges {
		ch.challengeCh <- challenge
	}
}

func (ch *Child) checkPendingEvents(events []challengertypes.ChallengeEvent) ([]challengertypes.Challenge, []types.RawKV, error) {
	challenges := make([]challengertypes.Challenge, 0)
	kvs, err := ch.PendingEventsToRawKV(ch.eventQueue, true)
	if err != nil {
		return nil, nil, err
	}

	for _, event := range events {
		pendingEvent, ok := ch.getPendingEvent(event.Id())
		if !ok {
			return nil, nil, errors.New("pending event not found")
		}

		ok, err := pendingEvent.Equal(event)
		if err != nil {
			return nil, nil, err
		} else if !ok {
			challenges = append(challenges, challengertypes.Challenge{
				Id:   event.Id(),
				Log:  fmt.Sprintf("pending event does not match; expected: %s, got: %s", pendingEvent.String(), event.String()),
				Time: event.EventTime(),
			})
		} else {
			ch.Logger().Info("pending event matched", zap.String("event", pendingEvent.String()))
		}

		if event.Type() == challengertypes.EventTypeOracle {
			oracleEvents := ch.getOraclePendingEvents(event.Id().Id)
			oracleKVs, err := ch.PendingEventsToRawKV(oracleEvents, true)
			if err != nil {
				return nil, nil, err
			}
			kvs = append(kvs, oracleKVs...)
		}
	}

	for _, challenge := range challenges {
		value, err := challenge.Marshal()
		if err != nil {
			return nil, nil, err
		}

		kvs = append(kvs, types.RawKV{
			Key:   ch.DB().PrefixedKey(challengertypes.PrefixedPendingChallenge(challenge.Id)),
			Value: value,
		})
	}

	return challenges, kvs, nil
}

func (ch *Child) getPendingEvent(id challengertypes.ChallengeId) (challengertypes.ChallengeEvent, bool) {
	ch.pendingEventsMu.Lock()
	defer ch.pendingEventsMu.Unlock()

	event, ok := ch.pendingEvents[id]
	return event, ok
}

func (ch *Child) deletePendingEvent(id challengertypes.ChallengeId) {
	ch.pendingEventsMu.Lock()
	defer ch.pendingEventsMu.Unlock()

	delete(ch.pendingEvents, id)
}

func (ch *Child) getOraclePendingEvents(l1BlockHeight uint64) []challengertypes.ChallengeEvent {
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

func (ch *Child) GetAllPendingEvents() []challengertypes.ChallengeEvent {
	ch.pendingEventsMu.Lock()
	defer ch.pendingEventsMu.Unlock()

	events := make([]challengertypes.ChallengeEvent, 0, len(ch.pendingEvents))

	for _, event := range ch.pendingEvents {
		events = append(events, event)
	}
	sort.Slice(events, func(i, j int) bool {
		if events[i].Type() == events[j].Type() {
			return events[i].Id().Id < events[j].Id().Id
		}
		return events[i].Type() < events[j].Type()
	})
	return events
}

func (ch *Child) SetPendingEvents(events []challengertypes.ChallengeEvent) {
	ch.pendingEventsMu.Lock()
	defer ch.pendingEventsMu.Unlock()

	for _, event := range events {
		ch.pendingEvents[event.Id()] = event
	}
}

func (ch *Child) loadPendingEvents() (events []challengertypes.ChallengeEvent, err error) {
	iterErr := ch.DB().PrefixedIterate(challengertypes.PendingEventKey, func(_, value []byte) (stop bool, err error) {
		var event challengertypes.ChallengeEvent
		err = json.Unmarshal(value, &event)
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
