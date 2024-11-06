package host

import (
	"errors"
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

func (h Host) GetStatus() (Status, error) {
	nodeStatus, err := h.GetNodeStatus()
	if err != nil {
		return Status{}, err
	}
	if h.eventHandler == nil {
		return Status{}, errors.New("event handler is not initialized")
	}

	return Status{
		Node:             nodeStatus,
		NumPendingEvents: h.eventHandler.NumPendingEvents(),
	}, nil
}

func (h Host) GetNodeStatus() (nodetypes.Status, error) {
	if h.Node() == nil {
		return nodetypes.Status{}, errors.New("node is not initialized")
	}
	return h.Node().GetStatus(), nil
}

func (h Host) GetAllPendingEvents() ([]challengertypes.ChallengeEvent, error) {
	if h.eventHandler == nil {
		return nil, errors.New("event handler is not initialized")
	}
	return h.eventHandler.GetAllSortedPendingEvents(), nil
}
