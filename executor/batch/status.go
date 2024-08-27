package batch

import (
	"time"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
)

type Status struct {
	Node                    nodetypes.Status      `json:"node"`
	BatchInfo               ophosttypes.BatchInfo `json:"batch_info"`
	CurrentBatchFileSize    int64                 `json:"current_batch_file_size"`
	LastBatchEndBlockNumber uint64                `json:"last_batch_end_block_number"`
	LastBatchSubmissionTime time.Time             `json:"last_batch_submission_time"`
}

func (bs BatchSubmitter) GetStatus() Status {
	fileSize, _ := bs.batchFileSize()

	return Status{
		Node:                    bs.node.GetStatus(),
		BatchInfo:               bs.BatchInfo().BatchInfo,
		CurrentBatchFileSize:    fileSize,
		LastBatchEndBlockNumber: bs.LastBatchEndBlockNumber,
		LastBatchSubmissionTime: bs.lastSubmissionTime,
	}
}
