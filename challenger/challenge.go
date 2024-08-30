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
		case elem := <-c.elemCh:
			c.registerElem(elem)
			challenges, err := c.handleElem(elem.ChallengeId())
			if err != nil {
				return err
			}

			kvs := make([]types.RawKV, 0)
			for _, challenge := range challenges {
				kv, err := c.saveChallenge(challenge)
				if err != nil {
					return err
				}
				kvs = append(kvs, kv)
				elemKvs, err := c.deleteElems(challenge.Id)
				if err != nil {
					return err
				}
				kvs = append(kvs, elemKvs...)
			}

			err = c.db.RawBatchSet(kvs...)
			if err != nil {
				return err
			}

			for _, challenge := range challenges {
				err = c.handleChallenge(challenge)
				if err != nil {
					return err
				}
			}
		}
	}
}

func (c *Challenger) registerElem(elem challengertypes.ChallengeElem) {
	if c.elems[elem.ChallengeId()] == nil {
		c.elems[elem.ChallengeId()] = make(map[challengertypes.NodeType]challengertypes.ChallengeElem)
	}
	c.elems[elem.ChallengeId()][elem.Node] = elem
}

func (c *Challenger) handleElem(cid challengertypes.ChallengeId) ([]challengertypes.Challenge, error) {
	if err := cid.Type.Validate(); err != nil {
		return nil, err
	}
	return c.checkValue(cid)
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

func (c *Challenger) deleteElems(cid challengertypes.ChallengeId) ([]types.RawKV, error) {
	kvs := []types.RawKV{}
	for _, elem := range c.elems[cid] {
		kvs = append(kvs, types.RawKV{
			Key:   c.db.PrefixedKey(challengertypes.PrefixedChallengeElem(elem)),
			Value: nil,
		})
	}
	return kvs, nil
}

func (c *Challenger) handleChallenge(challenge challengertypes.Challenge) error {
	// TODO: warning log or send to alerting system
	c.logger.Error("challenge", zap.Any("challenge", challenge))
	return nil
}
