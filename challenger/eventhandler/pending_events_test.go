package eventhandler

import (
	"testing"
	"time"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	"github.com/initia-labs/opinit-bots/db"
	"github.com/stretchr/testify/require"
)

func TestPendingEvent(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)

	eventHandler := NewChallengeEventHandler(db)

	events := []challengertypes.ChallengeEvent{
		challengertypes.NewDeposit(1, 2, "from", "to", "l1Denom", "amount", time.Unix(0, 100).UTC()),
		challengertypes.NewDeposit(2, 2, "from", "to", "l1Denom", "amount", time.Unix(0, 101).UTC()),
		challengertypes.NewOracle(3, []byte("data"), time.Unix(0, 103).UTC()),
		challengertypes.NewOracle(4, []byte("data2"), time.Unix(0, 104).UTC()),
	}

	eventHandler.SetPendingEvents(events)
	require.Len(t, eventHandler.pendingEvents, 4)

	for _, event := range events {
		e, ok := eventHandler.GetPendingEvent(event.Id())
		require.True(t, ok)
		require.Equal(t, e, event)
	}

	pendingEvents := eventHandler.GetAllPendingEvents()
	require.Len(t, pendingEvents, 4)
	require.ElementsMatch(t, pendingEvents, events)

	processedEvents := []challengertypes.ChallengeEvent{
		challengertypes.NewDeposit(1, 2, "from", "to", "l1Denom", "amount", time.Unix(0, 100).UTC()),
		challengertypes.NewOracle(3, []byte("data"), time.Unix(0, 103).UTC()),
	}

	unprocessedEvents := eventHandler.GetUnprocessedPendingEvents(processedEvents)
	require.ElementsMatch(t, unprocessedEvents, []challengertypes.ChallengeEvent{
		challengertypes.NewDeposit(2, 2, "from", "to", "l1Denom", "amount", time.Unix(0, 101).UTC()),
		challengertypes.NewOracle(4, []byte("data2"), time.Unix(0, 104).UTC()),
	})

	sortedPendingEvents := eventHandler.GetAllSortedPendingEvents()
	require.Equal(t, sortedPendingEvents, []challengertypes.ChallengeEvent{
		challengertypes.NewDeposit(1, 2, "from", "to", "l1Denom", "amount", time.Unix(0, 100).UTC()),
		challengertypes.NewDeposit(2, 2, "from", "to", "l1Denom", "amount", time.Unix(0, 101).UTC()),
		challengertypes.NewOracle(3, []byte("data"), time.Unix(0, 103).UTC()),
		challengertypes.NewOracle(4, []byte("data2"), time.Unix(0, 104).UTC()),
	})

	eventHandler.DeletePendingEvent(events[0].Id())
	require.Len(t, eventHandler.pendingEvents, 3)

	numEvents := eventHandler.NumPendingEvents()
	require.Equal(t, map[string]int64{
		challengertypes.EventTypeDeposit.String(): 1,
		challengertypes.EventTypeOracle.String():  2,
	}, numEvents)

	oracleEvents := eventHandler.getOraclePendingEvents(5)
	require.Len(t, oracleEvents, 2)
	require.ElementsMatch(t, oracleEvents, []challengertypes.ChallengeEvent{
		challengertypes.NewOracle(3, []byte("data"), time.Unix(0, 103).UTC()),
		challengertypes.NewOracle(4, []byte("data2"), time.Unix(0, 104).UTC()),
	})

	eventHandler.DeletePendingEvents(events[1:])
	require.Len(t, eventHandler.pendingEvents, 0)
}
