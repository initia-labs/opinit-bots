package executor

import (
	"github.com/initia-labs/opinit-bots/executor/batch"
	"github.com/initia-labs/opinit-bots/executor/child"
	"github.com/initia-labs/opinit-bots/executor/host"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
)

type Status struct {
	BridgeId int64            `json:"bridge_id"`
	Host     host.Status      `json:"host,omitempty"`
	Child    child.Status     `json:"child,omitempty"`
	Batch    batch.Status     `json:"batch,omitempty"`
	DA       nodetypes.Status `json:"da,omitempty"`
}

func (ex Executor) GetStatus() Status {
	s := Status{
		BridgeId: ex.host.BridgeId(),
	}
	if ex.host != nil {
		s.Host = ex.host.GetStatus()
	}
	if ex.child != nil {
		s.Child = ex.child.GetStatus()
	}
	if ex.batch != nil {
		s.Batch = ex.batch.GetStatus()
		if ex.batch.DA() != nil {
			s.DA = ex.batch.DA().GetNodeStatus()
		}
	}
	return s
}
