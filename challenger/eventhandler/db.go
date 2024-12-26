package eventhandler

import (
	"cosmossdk.io/errors"
	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	"github.com/initia-labs/opinit-bots/types"
)

func GetPendingEvents(db types.DB, ids []challengertypes.ChallengeId) ([]challengertypes.ChallengeEvent, error) {
	events := make([]challengertypes.ChallengeEvent, 0)
	for _, id := range ids {
		event, err := GetPendingEvent(db, id)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

func GetPendingEvent(db types.BasicDB, id challengertypes.ChallengeId) (challengertypes.ChallengeEvent, error) {
	data, err := db.Get(challengertypes.PrefixedPendingEvent(id))
	if err != nil {
		return nil, err
	}

	event, err := challengertypes.UnmarshalChallengeEvent(id.Type, data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal challenge event")
	}
	return event, err
}

func SavePendingEvents(db types.BasicDB, events []challengertypes.ChallengeEvent) error {
	for _, event := range events {
		err := SavePendingEvent(db, event)
		if err != nil {
			return err
		}
	}
	return nil
}

func SavePendingEvent(db types.BasicDB, event challengertypes.ChallengeEvent) error {
	data, err := event.Marshal()
	if err != nil {
		return err
	}
	return db.Set(challengertypes.PrefixedPendingEvent(event.Id()), data)
}

func DeletePendingEvents(db types.BasicDB, events []challengertypes.ChallengeEvent) error {
	for _, event := range events {
		err := DeletePendingEvent(db, event)
		if err != nil {
			return err
		}
	}
	return nil
}

func DeletePendingEvent(db types.BasicDB, event challengertypes.ChallengeEvent) error {
	return db.Delete(challengertypes.PrefixedPendingEvent(event.Id()))
}

func DeleteAllPendingEvents(db types.DB) error {
	deletingKeys := make([][]byte, 0)
	iterErr := db.Iterate(challengertypes.PendingEventKey, nil, func(key []byte, _ []byte) (stop bool, err error) {
		deletingKeys = append(deletingKeys, key)
		return false, nil
	})
	if iterErr != nil {
		return iterErr
	}

	for _, key := range deletingKeys {
		err := db.Delete(key)
		if err != nil {
			return err
		}
	}
	return nil
}

func LoadPendingEvents(db types.DB) (events []challengertypes.ChallengeEvent, err error) {
	iterErr := db.Iterate(challengertypes.PendingEventKey, nil, func(key, value []byte) (stop bool, err error) {
		id, err := challengertypes.ParsePendingEvent(key)
		if err != nil {
			return true, errors.Wrap(err, "failed to parse pending event key")
		}

		event, err := challengertypes.UnmarshalChallengeEvent(id.Type, value)
		if err != nil {
			return true, errors.Wrap(err, "failed to unmarshal challenge event")
		}
		events = append(events, event)
		return false, nil
	})
	if iterErr != nil {
		return nil, iterErr
	}
	return
}
