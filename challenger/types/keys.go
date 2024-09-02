package types

import (
	dbtypes "github.com/initia-labs/opinit-bots/db/types"
)

var (
	// Keys
	PendingEventKey = []byte("pending_event")
	ChallengeKey    = []byte("challenge")
)

func PrefixedEventId(id uint64) []byte {
	return append(dbtypes.FromUint64Key(id), dbtypes.Splitter)
}

func PrefixedEventType(eventType EventType) []byte {
	return append([]byte{byte(eventType)}, dbtypes.Splitter)
}

func PrefixedEventTypeId(eventType EventType, id uint64) []byte {
	return append(PrefixedEventType(eventType), PrefixedEventId(id)...)
}

func PrefixedPendingEvent(id ChallengeId) []byte {
	return append(append(PendingEventKey, dbtypes.Splitter),
		PrefixedEventTypeId(id.Type, id.Id)...)
}

func PrefixedChallenge(id ChallengeId) []byte {
	return append(append(ChallengeKey, dbtypes.Splitter),
		PrefixedEventTypeId(id.Type, id.Id)...)
}
