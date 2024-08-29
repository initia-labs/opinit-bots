package types

import (
	dbtypes "github.com/initia-labs/opinit-bots/db/types"
)

var (
	// Keys
	ChallengeElemKey = []byte("challenge_elem")
	ChallengeKey     = []byte("challenge")
)

func PrefixedElemId(id uint64) []byte {
	return append(dbtypes.FromUint64Key(id), dbtypes.Splitter)
}

func PrefixedElemEventType(eventType EventType) []byte {
	return append([]byte{byte(eventType)}, dbtypes.Splitter)
}

func PrefixedElemEventTypeId(eventType EventType, id uint64) []byte {
	return append(PrefixedElemEventType(eventType), PrefixedElemId(id)...)
}

func PrefixedElemKeyEventTypeId(eventType EventType, id uint64) []byte {
	return append(append(ChallengeElemKey, dbtypes.Splitter),
		PrefixedElemEventTypeId(eventType, id)...)
}

func PrefixedChallengeElem(elem ChallengeElem) []byte {
	return append(
		PrefixedElemKeyEventTypeId(elem.Event.Type(), elem.Id),
		byte(elem.Node),
	)
}

func PrefixedChallenge(id ChallengeId) []byte {
	return append(append(ChallengeKey, dbtypes.Splitter),
		PrefixedElemEventTypeId(id.Type, id.Id)...)
}
