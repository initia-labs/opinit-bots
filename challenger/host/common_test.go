package host

import (
	"context"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type mockChild struct {
	db             types.DB
	height         int64
	nextL1Sequence uint64
	pendingEvents  []challengertypes.ChallengeEvent
}

func NewMockChild(db types.DB, height int64, nextL1Sequence uint64) *mockChild {
	return &mockChild{
		db:             db,
		height:         height,
		nextL1Sequence: nextL1Sequence,
		pendingEvents:  make([]challengertypes.ChallengeEvent, 0),
	}
}

func (m *mockChild) DB() types.DB {
	return m.db
}

func (m *mockChild) SetPendingEvents(events []challengertypes.ChallengeEvent) {
	m.pendingEvents = append(m.pendingEvents, events...)
}

func (m *mockChild) Height() int64 {
	return m.height
}

func (m *mockChild) QueryNextL1Sequence(ctx context.Context, height int64) (uint64, error) {
	if m.nextL1Sequence == 0 {
		return 0, errors.New("no next L1 sequence")
	}
	return m.nextL1Sequence, nil
}

var _ childNode = (*mockChild)(nil)

type mockChallenger struct {
	db                types.DB
	pendingChallenges []challengertypes.Challenge
}

func NewMockChallenger(db types.DB) *mockChallenger {
	return &mockChallenger{
		db:                db,
		pendingChallenges: make([]challengertypes.Challenge, 0),
	}
}

func (m *mockChallenger) DB() types.DB {
	return m.db
}

func (m *mockChallenger) SendPendingChallenges(challenges []challengertypes.Challenge) {
	m.pendingChallenges = append(m.pendingChallenges, challenges...)
}

var _ challenger = (*mockChallenger)(nil)

func logCapturer() (*zap.Logger, *observer.ObservedLogs) {
	core, logs := observer.New(zap.DebugLevel)
	return zap.New(core), logs
}
