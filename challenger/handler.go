package challenger

import (
	"context"
	"sort"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	"github.com/initia-labs/opinit-bots/types"
	"go.uber.org/zap"
)

func (c *Challenger) challengeHandler(ctx context.Context) error {
	defer close(c.challengeChStopped)
	for {
		select {
		case <-ctx.Done():
			return nil
		case challenge := <-c.challengeCh:
			kvs := make([]types.RawKV, 0)
			kv := c.deletePendingChallenge(challenge)
			kvs = append(kvs, kv)
			kv, err := c.saveChallenge(challenge)
			if err != nil {
				return err
			}
			kvs = append(kvs, kv)

			err = c.handleChallenge(challenge)
			if err != nil {
				return err
			}

			err = c.db.RawBatchSet(kvs...)
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
				return c.latestChallenges[i].Id.Type < c.latestChallenges[j].Id.Type
			}
			return c.latestChallenges[i].Id.Id < c.latestChallenges[j].Id.Id
		}
		return c.latestChallenges[i].Time.Before(c.latestChallenges[j].Time)
	})
	if len(c.latestChallenges) > 10 {
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

func (c *Challenger) handleChallenge(challenge challengertypes.Challenge) error {
	// TODO: warning log or send to alerting system
	c.logger.Error("challenge", zap.Any("challenge", challenge))

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
