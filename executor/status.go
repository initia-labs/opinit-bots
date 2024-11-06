package executor

import (
	"github.com/initia-labs/opinit-bots/executor/batch"
	"github.com/initia-labs/opinit-bots/executor/child"
	"github.com/initia-labs/opinit-bots/executor/host"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
)

type Status struct {
	BridgeId uint64           `json:"bridge_id"`
	Host     host.Status      `json:"host,omitempty"`
	Child    child.Status     `json:"child,omitempty"`
	Batch    batch.Status     `json:"batch,omitempty"`
	DA       nodetypes.Status `json:"da,omitempty"`
}

func (ex Executor) GetStatus() (Status, error) {
	var err error

	s := Status{}
	if ex.host != nil {
		s.BridgeId = ex.host.BridgeId()
		s.Host, err = ex.host.GetStatus()
		if err != nil {
			return Status{}, err
		}
	}
	if ex.child != nil {
		s.Child, err = ex.child.GetStatus()
		if err != nil {
			return Status{}, err
		}
	}
	if ex.batch != nil {
		s.Batch, err = ex.batch.GetStatus()
		if err != nil {
			return Status{}, err
		}
		if ex.batch.DA() != nil {
			s.DA, err = ex.batch.DA().GetNodeStatus()
			if err != nil {
				return Status{}, err
			}
		}
	}
	return s, nil
}
