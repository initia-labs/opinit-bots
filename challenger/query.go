package challenger

import challengertypes "github.com/initia-labs/opinit-bots/challenger/types"

func (c *Challenger) QueryChallenges(page uint64) (challenges []challengertypes.Challenge, err error) {
	i := uint64(0)
	iterErr := c.db.PrefixedIterate(challengertypes.ChallengeKey, func(_, value []byte) (stop bool, err error) {
		i++
		if i >= (page+1)*100 {
			return true, nil
		}
		if i < page*100 {
			return false, nil
		}
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
