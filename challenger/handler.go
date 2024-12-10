package challenger

import (
	"sort"

	challengerdb "github.com/initia-labs/opinit-bots/challenger/db"
	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	"github.com/initia-labs/opinit-bots/types"
	"go.uber.org/zap"
)

func (c *Challenger) challengeHandler(ctx types.Context) error {
	defer close(c.challengeChStopped)
	for {
		select {
		case <-ctx.Done():
			return nil
		case challenge := <-c.challengeCh:
			c.stage.Reset()
            // Remove the pending challenge that was stored by the client or host
			err := challengerdb.DeletePendingChallenge(c.stage, challenge)
			if err != nil {
				return err
			}
			err = challengerdb.SaveChallenge(c.stage, challenge)
			if err != nil {
				return err
			}

			err = c.handleChallenge(ctx, challenge)
			if err != nil {
				return err
			}

			err = c.stage.Commit()
			if err != nil {
				return err
			}
			c.insertLatestChallenges(challenge)
		}
	}
}

func (c *Challenger) insertLatestChallenges(challenge challengertypes.Challenge) {
	c.latestChallengesMu.Lock()
	defer c.latestChallengesMu.Unlock()

	c.latestChallenges = append(c.latestChallenges, challenge)
	sort.Slice(c.latestChallenges, func(i, j int) bool {
		if c.latestChallenges[i].Time.Equal(c.latestChallenges[j].Time) {
			if c.latestChallenges[i].Id.Type == c.latestChallenges[j].Id.Type {
				return c.latestChallenges[i].Id.Id < c.latestChallenges[j].Id.Id
			}
			return c.latestChallenges[i].Id.Type < c.latestChallenges[j].Id.Type
		}
		return c.latestChallenges[i].Time.Before(c.latestChallenges[j].Time)
	})
	if len(c.latestChallenges) > 5 {
		c.latestChallenges = c.latestChallenges[1:]
	}
}

func (c *Challenger) getLatestChallenges() []challengertypes.Challenge {
	c.latestChallengesMu.Lock()
	defer c.latestChallengesMu.Unlock()

	res := make([]challengertypes.Challenge, len(c.latestChallenges))
	copy(res, c.latestChallenges)
	return res
}

func (c *Challenger) handleChallenge(ctx types.Context, challenge challengertypes.Challenge) error {
	// TODO: warning log or send to alerting system
	ctx.Logger().Error("challenge", zap.Any("challenge", challenge))

	return nil
}

func (c *Challenger) SendPendingChallenges(challenges []challengertypes.Challenge) {
	for _, challenge := range challenges {
		select {
		case <-c.challengeChStopped:
			return
		case c.challengeCh <- challenge:
		}
	}
}
