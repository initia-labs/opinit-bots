package db

import (
	"testing"
	"time"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	"github.com/initia-labs/opinit-bots/db"
	"github.com/stretchr/testify/require"
)

func TestPendingChallenge(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)

	challenge := challengertypes.Challenge{
		Id: challengertypes.ChallengeId{
			Type: challengertypes.EventTypeDeposit,
			Id:   10,
		},
	}

	_, err = GetPendingChallenge(db, challenge.Id)
	require.Error(t, err)

	err = SavePendingChallenge(db, challenge)
	require.NoError(t, err)

	c, err := GetPendingChallenge(db, challenge.Id)
	require.NoError(t, err)

	require.Equal(t, challenge, c)

	err = DeletePendingChallenge(db, challenge)
	require.NoError(t, err)

	_, err = GetPendingChallenge(db, challenge.Id)
	require.Error(t, err)
}

func TestPendingChallenges(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)

	var challenges []challengertypes.Challenge
	var ids []challengertypes.ChallengeId

	for i := uint64(0); i < 10; i++ {
		challenges = append(challenges, challengertypes.Challenge{
			Id: challengertypes.ChallengeId{
				Type: challengertypes.EventTypeOracle,
				Id:   i,
			},
		})
		ids = append(ids, challenges[i].Id)
	}

	_, err = GetPendingChallenges(db, ids)
	require.Error(t, err)

	err = SavePendingChallenges(db, challenges)
	require.NoError(t, err)

	c, err := GetPendingChallenges(db, ids)
	require.NoError(t, err)

	require.Equal(t, challenges, c)

	err = DeletePendingChallenges(db, challenges)
	require.NoError(t, err)

	_, err = GetPendingChallenges(db, ids)
	require.Error(t, err)
}

func TestChallenge(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)

	challenge := challengertypes.Challenge{
		Id: challengertypes.ChallengeId{
			Type: challengertypes.EventTypeDeposit,
			Id:   10,
		},
		Time: time.Unix(0, 10000).UTC(),
	}

	challenges, err := LoadChallenges(db)
	require.NoError(t, err)

	require.Empty(t, challenges)

	err = SaveChallenge(db, challenge)
	require.NoError(t, err)

	challenges, err = LoadChallenges(db)
	require.NoError(t, err)

	require.Len(t, challenges, 1)
	require.Equal(t, challenge, challenges[0])
}

func TestDeleteFutureChallenges(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)

	for i := 1; i <= 10; i++ {
		challenge := challengertypes.Challenge{
			Id: challengertypes.ChallengeId{
				Type: challengertypes.EventTypeOracle,
				Id:   uint64(i),
			},
			Time: time.Unix(0, int64(i)).UTC(),
		}
		err = SaveChallenge(db, challenge)
		require.NoError(t, err)
	}

	err = DeleteFutureChallenges(db, time.Unix(0, 5).UTC())
	require.NoError(t, err)

	challenges, err := LoadChallenges(db)
	require.NoError(t, err)

	require.Len(t, challenges, 4)
	for i := 1; i <= 4; i++ {
		require.Equal(t, uint64(i), challenges[i-1].Id.Id)
	}

	err = DeleteFutureChallenges(db, time.Unix(0, 8).UTC())
	require.NoError(t, err)

	challenges, err = LoadChallenges(db)
	require.NoError(t, err)

	require.Len(t, challenges, 4)
	for i := 1; i <= 4; i++ {
		require.Equal(t, uint64(i), challenges[i-1].Id.Id)
	}

	err = DeleteFutureChallenges(db, time.Unix(0, 0).UTC())
	require.NoError(t, err)

	challenges, err = LoadChallenges(db)
	require.NoError(t, err)

	require.Len(t, challenges, 0)
}

func TestDeleteAllPendingChallenges(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)

	for i := 1; i <= 10; i++ {
		challenge := challengertypes.Challenge{
			Id: challengertypes.ChallengeId{
				Type: challengertypes.EventTypeOracle,
				Id:   uint64(i),
			},
		}
		err = SavePendingChallenge(db, challenge)
		require.NoError(t, err)
	}

	err = DeleteAllPendingChallenges(db)
	require.NoError(t, err)

	challenges, err := LoadPendingChallenges(db)
	require.NoError(t, err)

	require.Len(t, challenges, 0)
}
