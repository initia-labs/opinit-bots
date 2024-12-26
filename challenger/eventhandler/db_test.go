package eventhandler

import (
	"testing"
	"time"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	"github.com/initia-labs/opinit-bots/db"
	"github.com/stretchr/testify/require"
)

func TestDBPendingEvent(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)

	event := challengertypes.NewDeposit(1, 2, "from", "to", "l1Denom", "amount", time.Unix(0, 10000))

	_, err = GetPendingEvent(db, event.Id())
	require.Error(t, err)

	err = SavePendingEvent(db, event)
	require.NoError(t, err)

	e, err := GetPendingEvent(db, event.Id())
	require.NoError(t, err)

	depositEvent, ok := e.(*challengertypes.Deposit)
	require.True(t, ok)

	require.Equal(t, event, depositEvent)

	err = DeletePendingEvent(db, event)
	require.NoError(t, err)

	_, err = GetPendingEvent(db, event.Id())
	require.Error(t, err)
}

func TestDBPendingEvents(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)

	var events []challengertypes.ChallengeEvent
	var ids []challengertypes.ChallengeId

	for i := uint64(0); i < 10; i++ {
		event := challengertypes.NewDeposit(i, 2, "from", "to", "l1Denom", "amount", time.Unix(0, 10000))
		events = append(events, event)
		ids = append(ids, event.Id())
	}

	_, err = GetPendingEvents(db, ids)
	require.Error(t, err)

	err = SavePendingEvents(db, events)
	require.NoError(t, err)

	es, err := GetPendingEvents(db, ids)
	require.NoError(t, err)

	for i, e := range es {
		depositEvent, ok := e.(*challengertypes.Deposit)
		require.True(t, ok)
		require.Equal(t, events[i], depositEvent)
	}

	err = DeletePendingEvents(db, events[5:])
	require.NoError(t, err)

	es, err = LoadPendingEvents(db)
	require.NoError(t, err)

	require.Equal(t, events[:5], es)

	err = DeleteAllPendingEvents(db)
	require.NoError(t, err)

	es, err = LoadPendingEvents(db)
	require.NoError(t, err)

	require.Empty(t, es)
}

func TestDBLoadPendingEvents(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)

	events := []challengertypes.ChallengeEvent{
		challengertypes.NewDeposit(1, 2, "from", "to", "l1Denom", "amount", time.Unix(0, 100)),
		challengertypes.NewDeposit(2, 2, "from", "to", "l1Denom", "amount", time.Unix(0, 101)),
		challengertypes.NewOracle(1, []byte("data"), time.Unix(0, 102)),
	}

	err = SavePendingEvents(db, events)
	require.NoError(t, err)

	loadedEvents, err := LoadPendingEvents(db)
	require.NoError(t, err)

	require.Equal(t, events, loadedEvents)
}
