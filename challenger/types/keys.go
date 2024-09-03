package types

import (
	"time"

	dbtypes "github.com/initia-labs/opinit-bots/db/types"
)

var (
	// Keys
	PendingEventKey     = []byte("pending_event")
	PendingChallengeKey = []byte("pending_challenge")
	ChallengeKey        = []byte("challenge")
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

func PrefixedPendingChallenge(id ChallengeId) []byte {
	return append(append(PendingChallengeKey, dbtypes.Splitter),
		PrefixedEventTypeId(id.Type, id.Id)...)
}

func PrefixedTimeEvnetTypeId(eventTime time.Time, id ChallengeId) []byte {
	return append(append(dbtypes.FromUint64Key(uint64(eventTime.UnixNano())), dbtypes.Splitter),
		PrefixedEventTypeId(id.Type, id.Id)...)
}

func PrefixedChallenge(eventTime time.Time, id ChallengeId) []byte {
	return append(append(ChallengeKey, dbtypes.Splitter),
		PrefixedTimeEvnetTypeId(eventTime, id)...)
}
