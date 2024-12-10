package eventhandler

import (
	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	"github.com/initia-labs/opinit-bots/types"
)

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
