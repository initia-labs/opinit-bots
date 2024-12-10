package challenger

import (
	"encoding/base64"

	dbtypes "github.com/initia-labs/opinit-bots/db/types"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
)

func (c *Challenger) QueryChallenges(from string, limit uint64, descOrder bool) (res challengertypes.QueryChallengesResponse, err error) {
	challenges := []challengertypes.Challenge{}
	next := ""

	count := uint64(0)
	fetchFn := func(key, value []byte) (bool, error) {
		challenge := challengertypes.Challenge{}
		err = challenge.Unmarshal(value)
		if err != nil {
			return true, err
		}
		if count >= limit {
			next = base64.StdEncoding.EncodeToString(challengertypes.PrefixedChallenge(challenge.Time, challenge.Id))
			return true, nil
		}
		challenges = append(challenges, challenge)
		count++
		return false, nil
	}

	var startKey []byte
	if next != "" {
		startKey, err = base64.StdEncoding.DecodeString(from)
		if err != nil {
			return challengertypes.QueryChallengesResponse{}, err
		}
	}

	if descOrder {
		err = c.db.ReverseIterate(dbtypes.AppendSplitter(challengertypes.ChallengeKey), startKey, fetchFn)
		if err != nil {
			return challengertypes.QueryChallengesResponse{}, err
		}
	} else {
		err := c.db.Iterate(dbtypes.AppendSplitter(challengertypes.ChallengeKey), startKey, fetchFn)
		if err != nil {
			return challengertypes.QueryChallengesResponse{}, err
		}
	}

	if next != "" {
		res.Next = &next
	}
	return res, nil
}
