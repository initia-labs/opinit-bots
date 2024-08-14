package executor

import (
	"github.com/initia-labs/opinit-bots-go/executor/batch"
	"github.com/initia-labs/opinit-bots-go/executor/child"
	"github.com/initia-labs/opinit-bots-go/executor/host"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
)

type Status struct {
	BridgeId int64            `json:"bridge_id"`
	Host     host.Status      `json:"host"`
	Child    child.Status     `json:"child"`
	Batch    batch.Status     `json:"batch"`
	DA       nodetypes.Status `json:"da"`
}

func (e Executor) GetStatus() Status {
	return Status{
		BridgeId: e.host.BridgeId(),
		Host:     e.host.GetStatus(),
		Child:    e.child.GetStatus(),
		Batch:    e.batch.GetStatus(),
		DA:       e.batch.DA().GetNodeStatus(),
	}
}
