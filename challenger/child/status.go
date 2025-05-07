package child

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	dbtypes "github.com/initia-labs/opinit-bots/db/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"

	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	"github.com/initia-labs/opinit-bots/types"
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

	workingTree, err := ch.WorkingTree()
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

type InternalStatus struct {
	LastUpdatedOracleL1Height         int64     `json:"last_updated_oracle_height"`
	LastFinalizedDepositL1BlockHeight int64     `json:"last_finalized_deposit_l1_block_height"`
	LastFinalizedDepositL1Sequence    uint64    `json:"last_finalized_deposit_l1_sequence"`
	LastOutputSubmissionTime          time.Time `json:"last_output_submission_time"`
}

func (ch Child) GetInternalStatus() InternalStatus {
	return InternalStatus{
		LastUpdatedOracleL1Height:         ch.lastUpdatedOracleL1Height,
		LastFinalizedDepositL1BlockHeight: ch.lastFinalizedDepositL1BlockHeight,
		LastFinalizedDepositL1Sequence:    ch.lastFinalizedDepositL1Sequence,
		LastOutputSubmissionTime:          ch.lastOutputTime,
	}
}

func (ch Child) SaveInternalStatus(db types.BasicDB) error {
	internalStatusBytes, err := json.Marshal(ch.GetInternalStatus())
	if err != nil {
		return errors.Wrap(err, "failed to marshal internal status")
	}
	return db.Set(executortypes.InternalStatusKey, internalStatusBytes)
}

func (ch *Child) LoadInternalStatus() error {
	internalStatusBytes, err := ch.DB().Get(executortypes.InternalStatusKey)
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
	ch.lastUpdatedOracleL1Height = internalStatus.LastUpdatedOracleL1Height
	ch.lastFinalizedDepositL1BlockHeight = internalStatus.LastFinalizedDepositL1BlockHeight
	ch.lastFinalizedDepositL1Sequence = internalStatus.LastFinalizedDepositL1Sequence
	ch.lastOutputTime = internalStatus.LastOutputSubmissionTime
	return nil
}
