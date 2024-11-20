package child

import (
	"errors"
	"time"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
)

type Status struct {
	Node                              nodetypes.Status `json:"node"`
	LastUpdatedOracleL1Height         int64            `json:"last_updated_oracle_height"`
	LastFinalizedDepositL1BlockHeight int64            `json:"last_finalized_deposit_l1_block_height"`
	LastFinalizedDepositL1Sequence    uint64           `json:"last_finalized_deposit_l1_sequence"`
	LastWithdrawalL2Sequence          uint64           `json:"last_withdrawal_l2_sequence"`
	WorkingTreeIndex                  uint64           `json:"working_tree_index"`

	FinalizingBlockHeight    int64     `json:"finalizing_block_height"`
	LastOutputSubmissionTime time.Time `json:"last_output_submission_time"`
	NextOutputSubmissionTime time.Time `json:"next_output_submission_time"`

	NumPendingEvents map[string]int64 `json:"num_pending_events"`
}

func (ch Child) GetStatus() (Status, error) {
	node := ch.Node()
	if node == nil {
		return Status{}, errors.New("node is not initialized")
	}

	workingTree, err := ch.GetWorkingTree()
	if err != nil {
		return Status{}, err
	}

	if ch.eventHandler == nil {
		return Status{}, errors.New("event handler is not initialized")
	}

	return Status{
		Node:                              node.GetStatus(),
		LastUpdatedOracleL1Height:         ch.lastUpdatedOracleL1Height,
		LastFinalizedDepositL1BlockHeight: ch.lastFinalizedDepositL1BlockHeight,
		LastFinalizedDepositL1Sequence:    ch.lastFinalizedDepositL1Sequence,
		LastWithdrawalL2Sequence:          workingTree.LeafCount + workingTree.StartLeafIndex - 1,
		WorkingTreeIndex:                  workingTree.Index,
		FinalizingBlockHeight:             ch.finalizingBlockHeight,
		LastOutputSubmissionTime:          ch.lastOutputTime,
		NextOutputSubmissionTime:          ch.nextOutputTime,
		NumPendingEvents:                  ch.eventHandler.NumPendingEvents(),
	}, nil
}

func (ch Child) GetAllPendingEvents() ([]challengertypes.ChallengeEvent, error) {
	if ch.eventHandler == nil {
		return nil, errors.New("event handler is not initialized")
	}
	return ch.eventHandler.GetAllSortedPendingEvents(), nil
}
