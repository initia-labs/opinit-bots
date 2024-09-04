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

func (c Challenger) GetStatus() Status {
	s := Status{
		BridgeId: c.host.BridgeId(),
	}
	if c.host != nil {
		s.Host = c.host.GetStatus()
	}
	if c.child != nil {
		s.Child = c.child.GetStatus()
	}
	s.LatestChallenges = c.getLatestChallenges()
	return s
}
