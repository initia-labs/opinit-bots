package challenger

import (
	"github.com/initia-labs/opinit-bots/challenger/child"
	"github.com/initia-labs/opinit-bots/challenger/host"
	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
)

type Status struct {
	BridgeId         uint64                      `json:"bridge_id"`
	Host             host.Status                 `json:"host,omitempty"`
	Child            child.Status                `json:"child,omitempty"`
	LatestChallenges []challengertypes.Challenge `json:"latest_challenges"`
}

func (c Challenger) GetStatus() (Status, error) {
	var err error
	s := Status{
		BridgeId: c.host.BridgeId(),
	}
	if c.host != nil {
		s.Host, err = c.host.GetStatus()
		if err != nil {
			return Status{}, err
		}
	}
	if c.child != nil {
		s.Child, err = c.child.GetStatus()
		if err != nil {
			return Status{}, err
		}
	}
	s.LatestChallenges = c.getLatestChallenges()
	return s, nil
}
