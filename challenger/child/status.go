package child

import (
	"time"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
)

type Status struct {
	Node                              nodetypes.Status `json:"node"`
	LastUpdatedOracleL1Height         uint64           `json:"last_updated_oracle_height"`
	LastFinalizedDepositL1BlockHeight uint64           `json:"last_finalized_deposit_l1_block_height"`
	LastFinalizedDepositL1Sequence    uint64           `json:"last_finalized_deposit_l1_sequence"`
	LastWithdrawalL2Sequence          uint64           `json:"last_withdrawal_l2_sequence"`
	WorkingTreeIndex                  uint64           `json:"working_tree_index"`

	FinalizingBlockHeight    uint64    `json:"finalizing_block_height"`
	LastOutputSubmissionTime time.Time `json:"last_output_submission_time"`
	NextOutputSubmissionTime time.Time `json:"next_output_submission_time"`

	NumPendingEvents map[string]int64 `json:"num_pending_events"`
}

func (ch Child) GetStatus() Status {
	return Status{
		Node:                              ch.Node().GetStatus(),
		LastUpdatedOracleL1Height:         ch.lastUpdatedOracleL1Height,
		LastFinalizedDepositL1BlockHeight: ch.lastFinalizedDepositL1BlockHeight,
		LastFinalizedDepositL1Sequence:    ch.lastFinalizedDepositL1Sequence,
		LastWithdrawalL2Sequence:          ch.Merkle().GetWorkingTreeLeafCount() + ch.Merkle().GetStartLeafIndex() - 1,
		WorkingTreeIndex:                  ch.Merkle().GetWorkingTreeIndex(),
		FinalizingBlockHeight:             ch.finalizingBlockHeight,
		LastOutputSubmissionTime:          ch.lastOutputTime,
		NextOutputSubmissionTime:          ch.nextOutputTime,
		NumPendingEvents:                  ch.eventHandler.NumPendingEvents(),
	}
}

func (ch Child) GetAllPendingEvents() []challengertypes.ChallengeEvent {
	return ch.eventHandler.GetAllSortedPendingEvents()
}
