package challenger

import (
	"context"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	"github.com/initia-labs/opinit-bots/types"
	"go.uber.org/zap"
)

func (c *Challenger) challengeHandler(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case challenge := <-c.challengeCh:
			err := c.handleChallenge(challenge)
			if err != nil {
				return err
			}
		}
	}
}

func (c *Challenger) saveChallenge(challenge challengertypes.Challenge) (types.RawKV, error) {
	value, err := challenge.Marshal()
	if err != nil {
		return types.RawKV{}, err
	}
	return types.RawKV{
		Key:   c.db.PrefixedKey(challengertypes.PrefixedChallenge(challenge.Id)),
		Value: value,
	}, nil
}

func (c *Challenger) handleChallenge(challenge challengertypes.Challenge) error {
	// TODO: warning log or send to alerting system
	c.logger.Error("challenge", zap.Any("challenge", challenge))
	return nil
}
