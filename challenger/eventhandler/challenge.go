package eventhandler

import (
	"fmt"
	"time"

	"cosmossdk.io/errors"
	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"go.uber.org/zap"

	"github.com/initia-labs/opinit-bots/types"
)

func (ch *ChallengeEventHandler) CheckValue(ctx types.Context, events []challengertypes.ChallengeEvent) ([]challengertypes.Challenge, []challengertypes.ChallengeEvent, error) {
	challenges := make([]challengertypes.Challenge, 0)
	processedEvents := make([]challengertypes.ChallengeEvent, 0)

	for _, event := range events {
		pendingEvent, ok := ch.GetPendingEvent(event.Id())
		if !ok {
			// might not happened because child always syncs later than host.
			return nil, nil, errors.Wrap(nodetypes.ErrIgnoreAndTryLater, fmt.Sprintf("pending event not found: %s", event.String()))
		}

		ok, err := pendingEvent.Equal(event)
		if err != nil {
			return nil, nil, err
		} else if !ok {
			challenges = append(challenges, challengertypes.Challenge{
				EventType: event.Type().String(),
				Id:        event.Id(),
				Log:       fmt.Sprintf("pending event does not match; expected: %s, got: %s", pendingEvent.String(), event.String()),
				Time:      event.EventTime(),
			})
		} else {
			ctx.Logger().Info("pending event matched", zap.String("event", pendingEvent.String()))
		}
		processedEvents = append(processedEvents, pendingEvent)

		// clearing pending oracle events up to l1 height of the last oracle event
		if event.Type() == challengertypes.EventTypeOracle {
			oracleEvents := ch.getOraclePendingEvents(event.Id().Id)
			processedEvents = append(processedEvents, oracleEvents...)
		}
	}

	return challenges, processedEvents, nil
}

func (ch *ChallengeEventHandler) GetPrevPendingEvent(event challengertypes.ChallengeEvent) (challengertypes.ChallengeEvent, bool) {
	prevId := event.Id()
	prevId.Id--
	prevEvent, ok := ch.GetPendingEvent(prevId)
	return prevEvent, ok
}

func (ch *ChallengeEventHandler) CheckTimeout(blockTime time.Time, events []challengertypes.ChallengeEvent) ([]challengertypes.Challenge, []challengertypes.ChallengeEvent) {
	challenges := make([]challengertypes.Challenge, 0)
	processedEvents := make([]challengertypes.ChallengeEvent, 0)

	for _, pendingEvent := range events {
		timeout := pendingEvent.EventTime().Add(ch.timeoutDuration)
		if !pendingEvent.EventTime().IsZero() && !pendingEvent.IsTimeout() && blockTime.After(timeout) {
			challenges = append(challenges, challengertypes.Challenge{
				EventType: pendingEvent.Type().String(),
				Id:        pendingEvent.Id(),
				Log:       fmt.Sprintf("event timeout: %s", pendingEvent.String()),
				Time:      blockTime,
			})
			pendingEvent.SetTimeout()
			processedEvents = append(processedEvents, pendingEvent)
		}
	}
	return challenges, processedEvents
}
