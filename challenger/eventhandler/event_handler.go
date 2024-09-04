package eventhandler

import (
	"sync"
	"time"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	"github.com/initia-labs/opinit-bots/types"
	"go.uber.org/zap"
)

type ChallengeEventHandler struct {
	db              types.DB
	logger          *zap.Logger
	pendingEventsMu *sync.Mutex
	pendingEvents   map[challengertypes.ChallengeId]challengertypes.ChallengeEvent
	timeoutDuration time.Duration
}

func NewChallengeEventHandler(db types.DB, logger *zap.Logger) *ChallengeEventHandler {
	return &ChallengeEventHandler{
		db:              db,
		logger:          logger,
		pendingEventsMu: &sync.Mutex{},
		pendingEvents:   make(map[challengertypes.ChallengeId]challengertypes.ChallengeEvent),
	}
}

func (ch *ChallengeEventHandler) Initialize(timeoutDuration time.Duration) error {
	pendingEvents, err := ch.loadPendingEvents()
	if err != nil {
		return err
	}
	ch.SetPendingEvents(pendingEvents)
	ch.timeoutDuration = timeoutDuration
	return nil
}
