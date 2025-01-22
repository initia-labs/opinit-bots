package child

import (
	"context"
	"errors"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	"github.com/initia-labs/opinit-bots/types"
)

type mockHost struct {
	db            types.DB
	height        int64
	pendingEvents []challengertypes.ChallengeEvent
	syncedOutputs map[uint64]map[uint64]*ophosttypes.Output
}

func NewMockHost(db types.DB, height int64) *mockHost {
	return &mockHost{
		db:            db,
		height:        height,
		pendingEvents: make([]challengertypes.ChallengeEvent, 0),
		syncedOutputs: make(map[uint64]map[uint64]*ophosttypes.Output),
	}
}

func (m *mockHost) DB() types.DB {
	return m.db
}

func (m *mockHost) SetPendingEvents(events []challengertypes.ChallengeEvent) {
	m.pendingEvents = append(m.pendingEvents, events...)
}

func (m *mockHost) Height() int64 {
	return m.height
}

func (m *mockHost) SetSyncedOutput(bridgeId uint64, outputIndex uint64, syncedOutput *ophosttypes.Output) {
	if _, ok := m.syncedOutputs[bridgeId]; !ok {
		m.syncedOutputs[bridgeId] = make(map[uint64]*ophosttypes.Output)
	}
	m.syncedOutputs[bridgeId][outputIndex] = syncedOutput
}

func (m *mockHost) QuerySyncedOutput(ctx context.Context, bridgeId uint64, outputIndex uint64) (*ophosttypes.QueryOutputProposalResponse, error) {
	if _, ok := m.syncedOutputs[bridgeId]; !ok {
		return nil, errors.New("collections: not found")
	}

	if _, ok := m.syncedOutputs[bridgeId][outputIndex]; !ok {
		return nil, errors.New("collections: not found")
	}

	return &ophosttypes.QueryOutputProposalResponse{
		BridgeId:       bridgeId,
		OutputIndex:    outputIndex,
		OutputProposal: *m.syncedOutputs[bridgeId][outputIndex],
	}, nil
}

var _ hostNode = (*mockHost)(nil)

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
