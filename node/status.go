package node

import (
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
)

func (n Node) GetStatus() nodetypes.Status {
	s := nodetypes.Status{}
	if n.cfg.ProcessType != nodetypes.PROCESS_TYPE_ONLY_BROADCAST {
		s.LastBlockHeight = n.GetHeight()
	}

	if n.broadcaster != nil {
		broadcasterStatus := n.broadcaster.GetStatus()
		s.Broadcaster = &broadcasterStatus
	}
	return s
}
