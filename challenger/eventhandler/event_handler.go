package eventhandler

import (
	"sync"
	"time"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	"github.com/initia-labs/opinit-bots/types"
)

type ChallengeEventHandler struct {
	db              types.DB
	pendingEventsMu *sync.Mutex
	pendingEvents   map[challengertypes.ChallengeId]challengertypes.ChallengeEvent
	timeoutDuration time.Duration
}

func NewChallengeEventHandler(db types.DB) *ChallengeEventHandler {
	return &ChallengeEventHandler{
		db:              db,
		pendingEventsMu: &sync.Mutex{},
		pendingEvents:   make(map[challengertypes.ChallengeId]challengertypes.ChallengeEvent),
	}
}

func (ch *ChallengeEventHandler) Initialize(timeoutDuration time.Duration) error {
	pendingEvents, err := LoadPendingEvents(ch.db)
	if err != nil {
		return err
	}
	ch.SetPendingEvents(pendingEvents)
	ch.timeoutDuration = timeoutDuration
	return nil
}
