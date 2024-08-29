package host

import (
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
)

type Status struct {
	Node nodetypes.Status `json:"node"`
}

func (h Host) GetStatus() Status {
	return Status{
		Node: h.GetNodeStatus(),
	}
}

func (h Host) GetNodeStatus() nodetypes.Status {
	return h.Node().GetStatus()
}
