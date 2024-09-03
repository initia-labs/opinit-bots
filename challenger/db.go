package challenger

import (
	"slices"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	"github.com/initia-labs/opinit-bots/types"
)

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
