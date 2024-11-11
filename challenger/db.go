package challenger

import (
	"fmt"
	"slices"
	"time"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	"github.com/initia-labs/opinit-bots/node"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"
)

func (c *Challenger) PendingChallengeToRawKVs(challenges []challengertypes.Challenge, delete bool) ([]types.RawKV, error) {
	kvs := make([]types.RawKV, 0, len(challenges))
	for _, challenge := range challenges {
		var value []byte
		var err error

		if !delete {
			value, err = challenge.Marshal()
			if err != nil {
				return nil, err
			}
		}
		kvs = append(kvs, types.RawKV{
			Key:   c.db.PrefixedKey(challengertypes.PrefixedPendingChallenge(challenge.Id)),
			Value: value,
		})
	}
	return kvs, nil
}

func (c *Challenger) deletePendingChallenge(challenge challengertypes.Challenge) types.RawKV {
	return types.RawKV{
		Key:   c.db.PrefixedKey(challengertypes.PrefixedPendingChallenge(challenge.Id)),
		Value: nil,
	}
}

func (c *Challenger) loadPendingChallenges() (challenges []challengertypes.Challenge, err error) {
	iterErr := c.db.PrefixedIterate(challengertypes.PendingChallengeKey, nil, func(_, value []byte) (stop bool, err error) {
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

func (c *Challenger) saveChallenge(challenge challengertypes.Challenge) (types.RawKV, error) {
	value, err := challenge.Marshal()
	if err != nil {
		return types.RawKV{}, err
	}
	return types.RawKV{
		Key:   c.db.PrefixedKey(challengertypes.PrefixedChallenge(challenge.Time, challenge.Id)),
		Value: value,
	}, nil
}

func (c *Challenger) loadChallenges() (challenges []challengertypes.Challenge, err error) {
	iterErr := c.db.PrefixedReverseIterate(challengertypes.ChallengeKey, nil, func(_, value []byte) (stop bool, err error) {
		challenge := challengertypes.Challenge{}
		err = challenge.Unmarshal(value)
		if err != nil {
			return true, err
		}
		challenges = append(challenges, challenge)
		if len(challenges) >= 5 {
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

func (c *Challenger) DeleteFutureChallenges(initialBlockTime time.Time) error {
	deletingKeys := make([][]byte, 0)
	iterErr := c.db.PrefixedReverseIterate(challengertypes.ChallengeKey, nil, func(key []byte, _ []byte) (stop bool, err error) {
		ts, _, err := challengertypes.ParseChallenge(key)
		if err != nil {
			return true, err
		}
		if !ts.After(initialBlockTime) {
			return true, nil
		}

		deletingKeys = append(deletingKeys, key)
		return false, nil
	})
	if iterErr != nil {
		return iterErr
	}

	for _, key := range deletingKeys {
		err := c.db.Delete(key)
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

	if err := DeletePendingEvents(nodeDB); err != nil {
		return err
	}

	if err := DeletePendingChallenges(nodeDB); err != nil {
		return err
	}

	if err := node.DeleteSyncInfo(nodeDB); err != nil {
		return err
	}
	fmt.Printf("reset height to 0 for node %s\n", string(nodeDB.GetPrefix()))
	return nil
}

func DeletePendingEvents(db types.DB) error {
	deletingKeys := make([][]byte, 0)
	iterErr := db.PrefixedIterate(challengertypes.PendingEventKey, nil, func(key []byte, _ []byte) (stop bool, err error) {
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

func DeletePendingChallenges(db types.DB) error {
	deletingKeys := make([][]byte, 0)
	iterErr := db.PrefixedIterate(challengertypes.PendingChallengeKey, nil, func(key []byte, _ []byte) (stop bool, err error) {
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
