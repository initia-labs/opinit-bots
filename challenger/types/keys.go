package types

import (
	"time"

	dbtypes "github.com/initia-labs/opinit-bots/db/types"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"
)

var (
	// Keys
	PendingEventKey     = []byte("pending_event")
	PendingChallengeKey = []byte("pending_challenge")
	ChallengeKey        = []byte("challenge")
	StatusKey           = []byte("status")
)

func prefixedEventType(eventType EventType) []byte {
	return append([]byte{byte(eventType)}, dbtypes.Splitter)
}

func prefixedEventTypeId(eventType EventType, id uint64) []byte {
	return append(prefixedEventType(eventType), dbtypes.FromUint64Key(id)...)
}

func PrefixedPendingEvent(id ChallengeId) []byte {
	return append(append(PendingEventKey, dbtypes.Splitter),
		prefixedEventTypeId(id.Type, id.Id)...)
}

func PrefixedPendingChallenge(id ChallengeId) []byte {
	return append(append(PendingChallengeKey, dbtypes.Splitter),
		prefixedEventTypeId(id.Type, id.Id)...)
}

func prefixedTimeEvent(eventTime time.Time) []byte {
	return append(dbtypes.FromUint64Key(types.MustInt64ToUint64(eventTime.UnixNano())), dbtypes.Splitter)
}

func PrefixedChallengeEventTime(eventTime time.Time) []byte {
	return append(append(ChallengeKey, dbtypes.Splitter),
		prefixedTimeEvent(eventTime)...)
}

func PrefixedChallenge(eventTime time.Time, id ChallengeId) []byte {
	return append(PrefixedChallengeEventTime(eventTime),
		prefixedEventTypeId(id.Type, id.Id)...)
}

func ParsePendingEvent(key []byte) (ChallengeId, error) {
	if len(key) < 10 {
		return ChallengeId{}, errors.New("invalid key bytes")
	}

	cursor := len(key) - 10
	typeBz := key[cursor : cursor+1]
	cursor += 1 + 1 // u8 + splitter
	idBz := key[cursor:]
	return ChallengeId{Type: EventType(typeBz[0]), Id: dbtypes.ToUint64Key(idBz)}, nil
}

func ParseChallenge(key []byte) (time.Time, ChallengeId, error) {
	if len(key) < 19 {
		return time.Time{}, ChallengeId{}, errors.New("invalid key bytes")
	}

	cursor := len(key) - 19
	timeBz := key[cursor : cursor+8]
	cursor += 8 + 1 // u64 + splitter
	typeBz := key[cursor : cursor+1]
	cursor += 1 + 1 // u8 + splitter
	idBz := key[cursor:]
	return time.Unix(0, types.MustUint64ToInt64(dbtypes.ToUint64Key(timeBz))).UTC(), ChallengeId{Type: EventType(typeBz[0]), Id: dbtypes.ToUint64Key(idBz)}, nil
}
