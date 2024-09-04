package types

import (
	"time"

	dbtypes "github.com/initia-labs/opinit-bots/db/types"
	"github.com/pkg/errors"
)

var (
	// Keys
	PendingEventKey     = []byte("pending_event")
	PendingChallengeKey = []byte("pending_challenge")
	ChallengeKey        = []byte("challenge")
	StatusKey           = []byte("status")
)

func PrefixedEventType(eventType EventType) []byte {
	return append([]byte{byte(eventType)}, dbtypes.Splitter)
}

func PrefixedEventTypeId(eventType EventType, id uint64) []byte {
	return append(PrefixedEventType(eventType), dbtypes.FromUint64Key(id)...)
}

func PrefixedPendingEvent(id ChallengeId) []byte {
	return append(append(PendingEventKey, dbtypes.Splitter),
		PrefixedEventTypeId(id.Type, id.Id)...)
}

func PrefixedPendingChallenge(id ChallengeId) []byte {
	return append(append(PendingChallengeKey, dbtypes.Splitter),
		PrefixedEventTypeId(id.Type, id.Id)...)
}

func PrefixedTimeEventTypeId(eventTime time.Time, id ChallengeId) []byte {
	return append(append(dbtypes.FromUint64Key(uint64(eventTime.UnixNano())), dbtypes.Splitter),
		PrefixedEventTypeId(id.Type, id.Id)...)
}

func PrefixedChallenge(eventTime time.Time, id ChallengeId) []byte {
	return append(append(ChallengeKey, dbtypes.Splitter),
		PrefixedTimeEventTypeId(eventTime, id)...)
}

func ParsePendingEvent(key []byte) (ChallengeId, error) {
	if len(key) < 10 {
		return ChallengeId{}, errors.New("invalid key bytes")
	}

	typeBz := key[len(key)-10 : len(key)-9]
	idBz := key[len(key)-8:]
	return ChallengeId{Type: EventType(typeBz[0]), Id: dbtypes.ToUint64Key(idBz)}, nil
}
