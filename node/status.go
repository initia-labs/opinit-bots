package node

import (
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
)

func (n Node) GetStatus() nodetypes.Status {
	s := nodetypes.Status{}
	if n.cfg.ProcessType != nodetypes.PROCESS_TYPE_ONLY_BROADCAST {
		height := n.GetHeight()
		s.LastBlockHeight = &height
		s.Syncing = &n.syncing
	}

	if n.broadcaster != nil {
		broadcasterStatus := n.broadcaster.GetStatus()
		s.Broadcaster = &broadcasterStatus
	}
	return s
}
