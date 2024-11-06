package child

import (
	"errors"
	"time"

	nodetypes "github.com/initia-labs/opinit-bots/node/types"
)

type Status struct {
	Node                              nodetypes.Status `json:"node"`
	LastUpdatedOracleL1Height         int64            `json:"last_updated_oracle_height"`
	LastFinalizedDepositL1BlockHeight int64            `json:"last_finalized_deposit_l1_block_height"`
	LastFinalizedDepositL1Sequence    uint64           `json:"last_finalized_deposit_l1_sequence"`
	LastWithdrawalL2Sequence          uint64           `json:"last_withdrawal_l2_sequence"`
	WorkingTreeIndex                  uint64           `json:"working_tree_index"`
	// if it is not 0, we are syncing
	FinalizingBlockHeight    int64     `json:"finalizing_block_height"`
	LastOutputSubmissionTime time.Time `json:"last_output_submission_time"`
	NextOutputSubmissionTime time.Time `json:"next_output_submission_time"`
}

func (ch Child) GetStatus() (Status, error) {
	node := ch.Node()
	if node == nil {
		return Status{}, errors.New("node is not initialized")
	}

	workingTreeLeafCount, err := ch.GetWorkingTreeLeafCount()
	if err != nil {
		return Status{}, err
	}
	startLeafIndex, err := ch.GetStartLeafIndex()
	if err != nil {
		return Status{}, err
	}
	workingTreeIndex, err := ch.GetWorkingTreeIndex()
	if err != nil {
		return Status{}, err
	}

	return Status{
		Node:                              node.GetStatus(),
		LastUpdatedOracleL1Height:         ch.lastUpdatedOracleL1Height,
		LastFinalizedDepositL1BlockHeight: ch.lastFinalizedDepositL1BlockHeight,
		LastFinalizedDepositL1Sequence:    ch.lastFinalizedDepositL1Sequence,
		LastWithdrawalL2Sequence:          workingTreeLeafCount + startLeafIndex - 1,
		WorkingTreeIndex:                  workingTreeIndex,
		FinalizingBlockHeight:             ch.finalizingBlockHeight,
		LastOutputSubmissionTime:          ch.lastOutputTime,
		NextOutputSubmissionTime:          ch.nextOutputTime,
	}, nil
}
