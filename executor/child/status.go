package child

import (
	"time"

	nodetypes "github.com/initia-labs/opinit-bots/node/types"
)

type Status struct {
	Node                              nodetypes.Status `json:"node"`
	LastUpdatedOracleL1Height         uint64           `json:"last_updated_oracle_height"`
	LastFinalizedDepositL1BlockHeight uint64           `json:"last_finalized_deposit_l1_block_height"`
	LastFinalizedDepositL1Sequence    uint64           `json:"last_finalized_deposit_l1_sequence"`
	LastWithdrawalL2Sequence          uint64           `json:"last_withdrawal_l2_sequence"`
	WorkingTreeIndex                  uint64           `json:"working_tree_index"`
	// if it is not 0, we are syncing
	FinalizingBlockHeight    uint64    `json:"finalizing_block_height"`
	LastOutputSubmissionTime time.Time `json:"last_output_submission_time"`
	NextOutputSubmissionTime time.Time `json:"next_output_submission_time"`
}

func (ch Child) GetStatus() Status {
	return Status{
		Node:                              ch.node.GetStatus(),
		LastUpdatedOracleL1Height:         ch.lastUpdatedOracleL1Height,
		LastFinalizedDepositL1BlockHeight: ch.lastFinalizedDepositL1BlockHeight,
		LastFinalizedDepositL1Sequence:    ch.lastFinalizedDepositL1Sequence,
		LastWithdrawalL2Sequence:          ch.mk.GetWorkingTreeLeafCount() + ch.mk.GetStartLeafIndex() - 1,
		WorkingTreeIndex:                  ch.mk.GetWorkingTreeIndex(),
		FinalizingBlockHeight:             ch.finalizingBlockHeight,
		LastOutputSubmissionTime:          ch.lastOutputTime,
		NextOutputSubmissionTime:          ch.nextOutputTime,
	}
}
