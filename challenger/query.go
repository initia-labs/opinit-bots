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
		if count >= limit {
			next = base64.StdEncoding.EncodeToString(key)
			return true, nil
		}
		challenge := challengertypes.Challenge{}
		err = challenge.Unmarshal(value)
		if err != nil {
			return true, err
		}
		count++
		challenges = append(challenges, challenge)
		return false, nil
	}

	var startKey []byte
	if from != "" {
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
		err = c.db.Iterate(dbtypes.AppendSplitter(challengertypes.ChallengeKey), startKey, fetchFn)
		if err != nil {
			return challengertypes.QueryChallengesResponse{}, err
		}
	}

	if next != "" {
		res.Next = &next
	}
	res.Challenges = challenges
	return res, nil
}
