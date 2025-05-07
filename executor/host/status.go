package host

import (
	"encoding/json"
	"time"

	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"

	dbtypes "github.com/initia-labs/opinit-bots/db/types"
	executortypes "github.com/initia-labs/opinit-bots/executor/types"
)

type Status struct {
	Node                            nodetypes.Status `json:"node"`
	LastProposedOutputIndex         uint64           `json:"last_proposed_output_index"`
	LastProposedOutputL2BlockNumber int64            `json:"last_proposed_output_l2_block_number"`
	LastProposedOutputTime          time.Time        `json:"last_proposed_output_time"`
}

func (h Host) GetStatus() (Status, error) {
	nodeStatus, err := h.GetNodeStatus()
	if err != nil {
		return Status{}, errors.Wrap(err, "failed to get node status")
	}

	return Status{
		Node:                            nodeStatus,
		LastProposedOutputIndex:         h.lastProposedOutputIndex,
		LastProposedOutputL2BlockNumber: h.lastProposedOutputL2BlockNumber,
		LastProposedOutputTime:          h.lastProposedOutputTime,
	}, nil
}

func (h Host) GetNodeStatus() (nodetypes.Status, error) {
	if h.Node() == nil {
		return nodetypes.Status{}, errors.New("node is not initialized")
	}
	return h.Node().GetStatus(), nil
}

type InternalStatus struct {
	LastProposedOutputIndex         uint64    `json:"last_proposed_output_index"`
	LastProposedOutputL2BlockNumber int64     `json:"last_proposed_output_l2_block_number"`
	LastProposedOutputTime          time.Time `json:"last_proposed_output_time"`
	LastUpdatedBatchTime            time.Time `json:"last_updated_batch_time"`
}

func (h Host) GetInternalStatus() InternalStatus {
	return InternalStatus{
		LastProposedOutputIndex:         h.lastProposedOutputIndex,
		LastProposedOutputL2BlockNumber: h.lastProposedOutputL2BlockNumber,
		LastProposedOutputTime:          h.lastProposedOutputTime,
		LastUpdatedBatchTime:            h.lastUpdatedBatchTime,
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
	if errors.Is(err, dbtypes.ErrNotFound) {
		return nil
	} else if err != nil {
		return errors.Wrap(err, "failed to get internal status")
	}
	var internalStatus InternalStatus
	err = json.Unmarshal(internalStatusBytes, &internalStatus)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal internal status")
	}
	h.lastProposedOutputIndex = internalStatus.LastProposedOutputIndex
	h.lastProposedOutputL2BlockNumber = internalStatus.LastProposedOutputL2BlockNumber
	h.lastProposedOutputTime = internalStatus.LastProposedOutputTime
	h.lastUpdatedBatchTime = internalStatus.LastUpdatedBatchTime
	return nil
}
