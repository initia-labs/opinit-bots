package challenger

import (
	"context"

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
