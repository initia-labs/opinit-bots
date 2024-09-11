package host

import (
	"time"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
)

type Status struct {
	Node             nodetypes.Status `json:"node"`
	LastOutputIndex  uint64           `json:"last_output_index"`
	LastOutputTime   time.Time        `json:"last_output_time"`
	NumPendingEvents map[string]int64 `json:"num_pending_events"`
}

func (h Host) GetStatus() Status {
	return Status{
		Node:             h.GetNodeStatus(),
		NumPendingEvents: h.eventHandler.NumPendingEvents(),
	}
}

func (h Host) GetNodeStatus() nodetypes.Status {
	return h.Node().GetStatus()
}

func (h Host) GetAllPendingEvents() []challengertypes.ChallengeEvent {
	return h.eventHandler.GetAllSortedPendingEvents()
}
