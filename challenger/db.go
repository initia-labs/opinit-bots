package challenger

import (
	"fmt"
	"slices"

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
	iterErr := c.db.PrefixedIterate(challengertypes.PendingChallengeKey, func(_, value []byte) (stop bool, err error) {
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
	iterErr := c.db.PrefixedReverseIterate(challengertypes.ChallengeKey, func(_, value []byte) (stop bool, err error) {
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

func ResetHeights(db types.DB) error {
	dbs := []types.DB{
		db.WithPrefix([]byte(types.HostName)),
		db.WithPrefix([]byte(types.ChildName)),
	}

	for _, db := range dbs {
		if err := node.DeleteSyncInfo(db); err != nil {
			return err
		}
		fmt.Printf("reset height to 0 for node %s\n", string(db.GetPrefix()))
	}
	return nil
}

func ResetHeight(db types.DB, nodeName string) error {
	if nodeName != types.HostName &&
		nodeName != types.ChildName {
		return errors.New("unknown node name")
	}
	nodeDB := db.WithPrefix([]byte(nodeName))
	err := node.DeleteSyncInfo(nodeDB)
	if err != nil {
		return err
	}
	fmt.Printf("reset height to 0 for node %s\n", string(nodeDB.GetPrefix()))
	return nil
}
