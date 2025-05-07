package host

import (
	"encoding/json"
	"time"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/pkg/errors"

	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	"github.com/initia-labs/opinit-bots/types"
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
		return Status{}, errors.Wrap(err, "failed to get node status")
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

type InternalStatus struct {
	LastOutputIndex uint64
	LastOutputTime  time.Time
}

func (h Host) GetInternalStatus() InternalStatus {
	return InternalStatus{
		LastOutputIndex: h.lastOutputIndex,
		LastOutputTime:  h.lastOutputTime,
	}
}

func (h Host) SaveInternalStatus(db types.BasicDB) error {
	internalStatusBytes, err := json.Marshal(h.GetInternalStatus())
	if err != nil {
		return errors.Wrap(err, "failed to marshal internal status")
	}
	return db.Set(executortypes.InternalStatusKey, internalStatusBytes)
}

func (h *Host) LoadInternalStatus() error {
	internalStatusBytes, err := h.DB().Get(executortypes.InternalStatusKey)
	if err != nil {
		return errors.Wrap(err, "failed to get internal status")
	}
	var internalStatus InternalStatus
	err = json.Unmarshal(internalStatusBytes, &internalStatus)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal internal status")
	}
	h.lastOutputIndex = internalStatus.LastOutputIndex
	h.lastOutputTime = internalStatus.LastOutputTime
	return nil
}
