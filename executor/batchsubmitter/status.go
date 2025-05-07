package batchsubmitter

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"

	executortypes "github.com/initia-labs/opinit-bots/executor/types"
)

type Status struct {
	Node                    nodetypes.Status      `json:"node"`
	BatchInfo               ophosttypes.BatchInfo `json:"batch_info"`
	CurrentBatchSize        int64                 `json:"current_batch_size"`
	BatchStartBlockNumber   int64                 `json:"batch_start_block_number"`
	BatchEndBlockNumber     int64                 `json:"batch_end_block_number"`
	LastBatchSubmissionTime time.Time             `json:"last_batch_submission_time"`
}

func (bs BatchSubmitter) GetStatus() (Status, error) {
	if bs.node == nil {
		return Status{}, errors.New("node is not initialized")
	}

	if bs.BatchInfo() == nil {
		return Status{}, errors.New("batch info is not initialized")
	}

	return Status{
		Node:                    bs.node.GetStatus(),
		BatchInfo:               bs.BatchInfo().BatchInfo,
		CurrentBatchSize:        bs.localBatchInfo.BatchSize,
		BatchStartBlockNumber:   bs.localBatchInfo.Start,
		BatchEndBlockNumber:     bs.localBatchInfo.End,
		LastBatchSubmissionTime: bs.localBatchInfo.LastSubmissionTime,
	}, nil
}

type InternalStatus struct {
	LastBatchEndBlockNumber int64
}

func (bs BatchSubmitter) GetInternalStatus() InternalStatus {
	return InternalStatus{
		LastBatchEndBlockNumber: bs.LastBatchEndBlockNumber,
	}
}

func (bs BatchSubmitter) SaveInternalStatus(db types.BasicDB) error {
	internalStatusBytes, err := json.Marshal(bs.GetInternalStatus())
	if err != nil {
		return errors.Wrap(err, "failed to marshal internal status")
	}
	return db.Set(executortypes.InternalStatusKey, internalStatusBytes)
}

func (bs *BatchSubmitter) LoadInternalStatus() error {
	internalStatusBytes, err := bs.DB().Get(executortypes.InternalStatusKey)
	if err != nil {
		return errors.Wrap(err, "failed to get internal status")
	}
	var internalStatus InternalStatus
	err = json.Unmarshal(internalStatusBytes, &internalStatus)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal internal status")
	}
	bs.LastBatchEndBlockNumber = internalStatus.LastBatchEndBlockNumber
	return nil
}
