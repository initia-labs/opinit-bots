package db

import (
	"fmt"
	"slices"
	"time"

	"github.com/initia-labs/opinit-bots/challenger/eventhandler"
	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	"github.com/initia-labs/opinit-bots/node"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"
)

func GetPendingChallenge(db types.BasicDB, id challengertypes.ChallengeId) (challengertypes.Challenge, error) {
	data, err := db.Get(challengertypes.PrefixedPendingChallenge(id))
	if err != nil {
		return challengertypes.Challenge{}, err
	}
	challenge := challengertypes.Challenge{}
	err = challenge.Unmarshal(data)
	return challenge, err
}

func GetPendingChallenges(db types.DB, ids []challengertypes.ChallengeId) (challenges []challengertypes.Challenge, err error) {
	for _, id := range ids {
		challenge, err := GetPendingChallenge(db, id)
		if err != nil {
			return nil, err
		}
		challenges = append(challenges, challenge)
	}
	return
}

func SavePendingChallenge(db types.BasicDB, challenge challengertypes.Challenge) error {
	data, err := challenge.Marshal()
	if err != nil {
		return err
	}
	return db.Set(challengertypes.PrefixedPendingChallenge(challenge.Id), data)
}

func SavePendingChallenges(db types.BasicDB, challenges []challengertypes.Challenge) error {
	for _, challenge := range challenges {
		err := SavePendingChallenge(db, challenge)
		if err != nil {
			return err
		}
	}
	return nil
}

func DeletePendingChallenge(db types.BasicDB, challenge challengertypes.Challenge) error {
	return db.Delete(challengertypes.PrefixedPendingChallenge(challenge.Id))
}

func DeletePendingChallenges(db types.BasicDB, challenges []challengertypes.Challenge) error {
	for _, challenge := range challenges {
		err := DeletePendingChallenge(db, challenge)
		if err != nil {
			return err
		}
	}
	return nil
}

func DeleteAllPendingChallenges(db types.DB) error {
	deletingKeys := make([][]byte, 0)
	iterErr := db.Iterate(challengertypes.PendingChallengeKey, nil, func(key []byte, _ []byte) (stop bool, err error) {
		deletingKeys = append(deletingKeys, key)
		return false, nil
	})
	if iterErr != nil {
		return iterErr
	}

	for _, key := range deletingKeys {
		err := db.Delete(key)
		if err != nil {
			return err
		}
	}
	return nil
}

func LoadPendingChallenges(db types.DB) (challenges []challengertypes.Challenge, err error) {
	iterErr := db.Iterate(challengertypes.PendingChallengeKey, nil, func(_, value []byte) (stop bool, err error) {
		challenge := challengertypes.Challenge{}
		err = challenge.Unmarshal(value)
		if err != nil {
			return true, err
		}
		challenges = append(challenges, challenge)
		return false, nil
	})
	if iterErr != nil {
		return nil, iterErr
	}
	return
}

func SaveChallenge(db types.BasicDB, challenge challengertypes.Challenge) error {
	value, err := challenge.Marshal()
	if err != nil {
		return err
	}
	return db.Set(challengertypes.PrefixedChallenge(challenge.Time, challenge.Id), value)
}

func LoadChallenges(db types.DB, limit int) (challenges []challengertypes.Challenge, err error) {
	if limit < 0 {
		return nil, errors.New("limit must be non-negative")
	}

	iterErr := db.ReverseIterate(challengertypes.ChallengeKey, nil, func(_, value []byte) (stop bool, err error) {
		challenge := challengertypes.Challenge{}
		err = challenge.Unmarshal(value)
		if err != nil {
			return true, err
		}
		challenges = append(challenges, challenge)
		if limit != 0 && len(challenges) >= limit {
			return true, nil
		}
		return false, nil
	})
	if iterErr != nil {
		return nil, iterErr
	}
	slices.Reverse(challenges)
	return
}

func DeleteFutureChallenges(db types.DB, initialBlockTime time.Time) error {
	deletingKeys := make([][]byte, 0)
	iterErr := db.ReverseIterate(challengertypes.ChallengeKey, nil, func(key []byte, _ []byte) (stop bool, err error) {
		ts, _, err := challengertypes.ParseChallenge(key)
		if err != nil {
			return true, err
		}
		if ts.Before(initialBlockTime) {
			return true, nil
		}

		deletingKeys = append(deletingKeys, key)
		return false, nil
	})
	if iterErr != nil {
		return iterErr
	}

	for _, key := range deletingKeys {
		err := db.Delete(key)
		if err != nil {
			return err
		}
	}
	return nil
}

func ResetHeights(db types.DB) error {
	dbNames := []string{
		types.HostName,
		types.ChildName,
	}

	for _, dbName := range dbNames {
		if err := ResetHeight(db, dbName); err != nil {
			return err
		}
	}
	return nil
}

func ResetHeight(db types.DB, nodeName string) error {
	if nodeName != types.HostName &&
		nodeName != types.ChildName {
		return errors.New("unknown node name")
	}
	nodeDB := db.WithPrefix([]byte(nodeName))

	if err := eventhandler.DeleteAllPendingEvents(nodeDB); err != nil {
		return err
	}

	if err := DeleteAllPendingChallenges(nodeDB); err != nil {
		return err
	}

	if err := node.DeleteSyncedHeight(nodeDB); err != nil {
		return err
	}
	fmt.Printf("reset height to 0 for node %s\n", string(nodeDB.GetPrefix()))
	return nil
}
