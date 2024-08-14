package batch

import (
	"time"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
)

type Status struct {
	Node                    nodetypes.Status      `json:"node"`
	BatchInfo               ophosttypes.BatchInfo `json:"batch_info"`
	CurrentBatchFileSize    int64                 `json:"current_batch_file_size"`
	LastBatchEndBlockNumber uint64                `json:"last_batch_end_block_number"`
	LastBatchSubmissionTime time.Time             `json:"last_batch_submission_time"`
}

func (b BatchSubmitter) GetStatus() Status {
	fileSize, _ := b.batchFileSize()

	return Status{
		Node:                    b.node.GetStatus(),
		BatchInfo:               b.BatchInfo().BatchInfo,
		CurrentBatchFileSize:    fileSize,
		LastBatchEndBlockNumber: b.LastBatchEndBlockNumber,
		LastBatchSubmissionTime: b.lastSubmissionTime,
	}
}
