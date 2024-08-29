package challenger

import (
	"fmt"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	"go.uber.org/zap"
)

func (c *Challenger) checkValue(cid challengertypes.ChallengeId) ([]challengertypes.Challenge, error) {
	if len(c.elems[cid]) != 2 {
		return nil, nil
	}
	expected := c.elems[cid][challengertypes.NodeTypeHost].Event
	got := c.elems[cid][challengertypes.NodeTypeChild].Event

	ok, err := expected.Equal(got)
	if err != nil {
		return nil, err
	} else if !ok {
		log := fmt.Sprintf("expected %s, got %s", expected.String(), got.String())
		challenge := challengertypes.NewChallenge(cid, log)
		c.logger.Error("challenge generated", zap.String("id", cid.String()), zap.String("log", log))

		return []challengertypes.Challenge{challenge}, nil
	}

	return nil, nil
}
