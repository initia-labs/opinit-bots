package eventhandler

import (
	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	"github.com/initia-labs/opinit-bots/types"
)

func (ch *ChallengeEventHandler) PendingEventsToRawKV(events []challengertypes.ChallengeEvent, delete bool) ([]types.RawKV, error) {
	kvs := make([]types.RawKV, 0, len(events))
	for _, event := range events {
		var data []byte
		var err error

		if !delete {
			data, err = event.Marshal()
			if err != nil {
				return nil, err
			}
		}
		kvs = append(kvs, types.RawKV{
			Key:   ch.db.PrefixedKey(challengertypes.PrefixedPendingEvent(event.Id())),
			Value: data,
		})
	}
	return kvs, nil
}
