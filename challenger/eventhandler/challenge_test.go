package eventhandler

import (
	"context"
	"testing"
	"time"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	"github.com/initia-labs/opinit-bots/db"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCheckValue(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)

	events := []challengertypes.ChallengeEvent{
		challengertypes.NewDeposit(1, 2, "from", "to", "l1Denom", "amount", time.Unix(0, 100)),
		challengertypes.NewDeposit(2, 2, "from", "to", "l1Denom", "amount", time.Unix(0, 101)),
		challengertypes.NewOracle(3, []byte("data"), time.Unix(0, 103)),
		challengertypes.NewOracle(4, []byte("data2"), time.Unix(0, 104)),
		challengertypes.NewOutput(5, 2, []byte("output"), time.Unix(0, 105)),
	}

	eventHandler := NewChallengeEventHandler(db)
	eventHandler.SetPendingEvents(events)

	checkingEvents := []challengertypes.ChallengeEvent{
		challengertypes.NewDeposit(1, 2, "from", "to", "l1Denom", "amount", time.Unix(0, 100)),
		challengertypes.NewOracle(4, []byte("wrongdata"), time.Unix(0, 103)),
		challengertypes.NewOutput(5, 2, []byte("wrongoutput"), time.Unix(0, 105)),
	}

	ctx := types.NewContext(context.Background(), zap.NewNop(), "")

	challenges, processedEvents, err := eventHandler.CheckValue(ctx, checkingEvents)
	require.NoError(t, err)

	require.Len(t, challenges, 2)
	require.Len(t, processedEvents, 4)

	require.Equal(t, challenges[0].Id, challengertypes.ChallengeId{Type: challengertypes.EventTypeOracle, Id: 4})
	require.Equal(t, challenges[1].Id, challengertypes.ChallengeId{Type: challengertypes.EventTypeOutput, Id: 2})

	require.ElementsMatch(t, processedEvents, append(events[:1], events[2:]...))
}

func TestGetPrevPendingEvent(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)

	events := []challengertypes.ChallengeEvent{
		challengertypes.NewDeposit(1, 2, "from", "to", "l1Denom", "amount", time.Unix(0, 100)),
		challengertypes.NewDeposit(2, 2, "from", "to", "l1Denom", "amount", time.Unix(0, 101)),
		challengertypes.NewOracle(3, []byte("data"), time.Unix(0, 103)),
		challengertypes.NewOracle(4, []byte("data2"), time.Unix(0, 104)),
		challengertypes.NewOutput(5, 2, []byte("output"), time.Unix(0, 105)),
	}

	eventHandler := NewChallengeEventHandler(db)
	eventHandler.SetPendingEvents(events)

	_, ok := eventHandler.GetPrevPendingEvent(events[0])
	require.False(t, ok)

	prevEvent, ok := eventHandler.GetPrevPendingEvent(events[1])
	require.True(t, ok)
	require.Equal(t, prevEvent, events[0])

	_, ok = eventHandler.GetPrevPendingEvent(events[2])
	require.False(t, ok)

	prevEvent, ok = eventHandler.GetPrevPendingEvent(events[3])
	require.True(t, ok)
	require.Equal(t, prevEvent, events[2])

	_, ok = eventHandler.GetPrevPendingEvent(events[4])
	require.False(t, ok)
}

func TestCheckTimeout(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)

	events := []challengertypes.ChallengeEvent{
		challengertypes.NewDeposit(1, 2, "from", "to", "l1Denom", "amount", time.Unix(0, 100)),
		challengertypes.NewDeposit(2, 2, "from", "to", "l1Denom", "amount", time.Unix(0, 101)),
		challengertypes.NewOracle(3, []byte("data"), time.Unix(0, 103)),
		challengertypes.NewOracle(4, []byte("data2"), time.Unix(0, 104)),
		challengertypes.NewOutput(5, 2, []byte("output"), time.Unix(0, 105)),
	}

	eventHandler := NewChallengeEventHandler(db)
	eventHandler.timeoutDuration = 2
	eventHandler.SetPendingEvents(events)

	blockTime := time.Unix(0, 106)

	challenges, processedEvents := eventHandler.CheckTimeout(blockTime, events)
	require.Len(t, challenges, 3)
	require.Len(t, processedEvents, 3)

	require.Equal(t, challenges[0].Id, challengertypes.ChallengeId{Type: challengertypes.EventTypeDeposit, Id: 1})
	require.Equal(t, challenges[1].Id, challengertypes.ChallengeId{Type: challengertypes.EventTypeDeposit, Id: 2})
	require.Equal(t, challenges[2].Id, challengertypes.ChallengeId{Type: challengertypes.EventTypeOracle, Id: 3})

	require.Equal(t, processedEvents[0].Id(), challengertypes.ChallengeId{Type: challengertypes.EventTypeDeposit, Id: 1})
	require.Equal(t, processedEvents[1].Id(), challengertypes.ChallengeId{Type: challengertypes.EventTypeDeposit, Id: 2})
	require.Equal(t, processedEvents[2].Id(), challengertypes.ChallengeId{Type: challengertypes.EventTypeOracle, Id: 3})
}

func TestCheckValueOracleRelay(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)

	events := []challengertypes.ChallengeEvent{
		challengertypes.NewOracleRelay(100, []byte("hash100"), time.Unix(0, 100)),
		challengertypes.NewOracleRelay(101, []byte("hash101"), time.Unix(0, 101)),
		challengertypes.NewOracleRelay(102, []byte("hash102"), time.Unix(0, 102)),
		challengertypes.NewOracleRelay(103, []byte("hash103"), time.Unix(0, 103)),
		challengertypes.NewOracleRelay(104, []byte("hash104"), time.Unix(0, 104)),
	}

	eventHandler := NewChallengeEventHandler(db)
	eventHandler.SetPendingEvents(events)

	checkingEvents := []challengertypes.ChallengeEvent{
		challengertypes.NewOracleRelay(104, []byte("hash104"), time.Unix(0, 104)),
	}

	ctx := types.NewContext(context.Background(), zap.NewNop(), "")

	challenges, processedEvents, err := eventHandler.CheckValue(ctx, checkingEvents)
	require.NoError(t, err)

	// no challenges because hash matches
	require.Len(t, challenges, 0)
	// processed: 104 (matched) + 100, 101, 102, 103 (cleared)
	require.Len(t, processedEvents, 5)

	// verify all events are marked as processed
	require.ElementsMatch(t, processedEvents, events)
}

func TestCheckValueOracleRelayMismatch(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)

	events := []challengertypes.ChallengeEvent{
		challengertypes.NewOracleRelay(100, []byte("hash100"), time.Unix(0, 100)),
		challengertypes.NewOracleRelay(101, []byte("hash101"), time.Unix(0, 101)),
	}

	eventHandler := NewChallengeEventHandler(db)
	eventHandler.SetPendingEvents(events)

	// L2 receives relay with wrong hash
	checkingEvents := []challengertypes.ChallengeEvent{
		challengertypes.NewOracleRelay(101, []byte("wrong_hash"), time.Unix(0, 101)),
	}

	ctx := types.NewContext(context.Background(), zap.NewNop(), "")

	challenges, processedEvents, err := eventHandler.CheckValue(ctx, checkingEvents)
	require.NoError(t, err)

	// one challenge for mismatched hash
	require.Len(t, challenges, 1)
	require.Equal(t, challenges[0].Id, challengertypes.ChallengeId{Type: challengertypes.EventTypeOracleRelay, Id: 101})

	// processed: 101 (checked) + 100 (cleared)
	require.Len(t, processedEvents, 2)
}

func TestCheckTimeoutOracleRelay(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)

	events := []challengertypes.ChallengeEvent{
		challengertypes.NewOracleRelay(100, []byte("hash100"), time.Unix(0, 100)),
		challengertypes.NewOracleRelay(101, []byte("hash101"), time.Unix(0, 101)),
		challengertypes.NewOracleRelay(105, []byte("hash105"), time.Unix(0, 105)),
	}

	eventHandler := NewChallengeEventHandler(db)
	eventHandler.timeoutDuration = 2
	eventHandler.SetPendingEvents(events)

	blockTime := time.Unix(0, 106)

	challenges, processedEvents := eventHandler.CheckTimeout(blockTime, events)

	// events at 100, 101 should timeout (106 > 100+2 and 106 > 101+2)
	// event at 105 should not timeout (106 < 105+2)
	require.Len(t, challenges, 2)
	require.Len(t, processedEvents, 2)

	require.Equal(t, challenges[0].Id, challengertypes.ChallengeId{Type: challengertypes.EventTypeOracleRelay, Id: 100})
	require.Equal(t, challenges[1].Id, challengertypes.ChallengeId{Type: challengertypes.EventTypeOracleRelay, Id: 101})
}
