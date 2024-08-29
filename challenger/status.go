package challenger

import (
	"github.com/initia-labs/opinit-bots/challenger/child"
	"github.com/initia-labs/opinit-bots/challenger/host"
)

type Status struct {
	BridgeId int64        `json:"bridge_id"`
	Host     host.Status  `json:"host,omitempty"`
	Child    child.Status `json:"child,omitempty"`
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
	return s
}
