package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	CustomRegistry = prometheus.NewRegistry()
	factory        = promauto.With(CustomRegistry)

	// bridge and operational metrics
	BridgeID = factory.NewGauge(prometheus.GaugeOpts{
		Name: "opinit_bridge_id",
		Help: "Bridge ID",
	})

	HostSyncing = factory.NewGauge(prometheus.GaugeOpts{
		Name: "opinit_host_syncing",
		Help: "Host node syncing status (1 = syncing, 0 = synced)",
	})

	HostLastBlockHeight = factory.NewGauge(prometheus.GaugeOpts{
		Name: "opinit_host_last_block_height",
		Help: "Host node last block height",
	})

	HostPendingTxs = factory.NewGauge(prometheus.GaugeOpts{
		Name: "opinit_host_pending_txs",
		Help: "Host node pending transactions",
	})

	HostAccountSequence = factory.NewGauge(prometheus.GaugeOpts{
		Name: "opinit_host_account_sequence",
		Help: "Host account sequence number",
	})

	HostLastProposedOutputIndex = factory.NewGauge(prometheus.GaugeOpts{
		Name: "opinit_host_last_proposed_output_index",
		Help: "Host last proposed output index",
	})

	HostLastProposedOutputL2Block = factory.NewGauge(prometheus.GaugeOpts{
		Name: "opinit_host_last_proposed_output_l2_block",
		Help: "Host last proposed output L2 block number",
	})

	HostLastProposedOutputTime = factory.NewGauge(prometheus.GaugeOpts{
		Name: "opinit_host_last_proposed_output_time",
		Help: "Host last proposed output time (unix timestamp)",
	})

	ChildSyncing = factory.NewGauge(prometheus.GaugeOpts{
		Name: "opinit_child_syncing",
		Help: "Child node syncing status (1 = syncing, 0 = synced)",
	})

	ChildLastBlockHeight = factory.NewGauge(prometheus.GaugeOpts{
		Name: "opinit_child_last_block_height",
		Help: "Child node last block height",
	})

	ChildPendingTxs = factory.NewGauge(prometheus.GaugeOpts{
		Name: "opinit_child_pending_txs",
		Help: "Child node pending transactions",
	})

	ChildLastUpdatedOracleHeight = factory.NewGauge(prometheus.GaugeOpts{
		Name: "opinit_child_last_updated_oracle_height",
		Help: "Child last updated oracle L1 height",
	})

	ChildLastFinalizedDepositL1BlockHeight = factory.NewGauge(prometheus.GaugeOpts{
		Name: "opinit_child_last_finalized_deposit_l1_block_height",
		Help: "Child last finalized deposit L1 block height",
	})

	ChildLastFinalizedDepositL1Sequence = factory.NewGauge(prometheus.GaugeOpts{
		Name: "opinit_child_last_finalized_deposit_l1_sequence",
		Help: "Child last finalized deposit L1 sequence",
	})

	ChildLastWithdrawalL2Sequence = factory.NewGauge(prometheus.GaugeOpts{
		Name: "opinit_child_last_withdrawal_l2_sequence",
		Help: "Child last withdrawal L2 sequence",
	})

	ChildWorkingTreeIndex = factory.NewGauge(prometheus.GaugeOpts{
		Name: "opinit_child_working_tree_index",
		Help: "Child working tree index",
	})

	ChildFinalizingBlockHeight = factory.NewGauge(prometheus.GaugeOpts{
		Name: "opinit_child_finalizing_block_height",
		Help: "Child finalizing block height (0 if not syncing)",
	})

	ChildLastOutputSubmissionTime = factory.NewGauge(prometheus.GaugeOpts{
		Name: "opinit_child_last_output_submission_time",
		Help: "Child last output submission time (unix timestamp)",
	})

	ChildNextOutputSubmissionTime = factory.NewGauge(prometheus.GaugeOpts{
		Name: "opinit_child_next_output_submission_time",
		Help: "Child next output submission time (unix timestamp)",
	})

	BatchSubmitterSyncing = factory.NewGauge(prometheus.GaugeOpts{
		Name: "opinit_batch_submitter_syncing",
		Help: "Batch submitter node syncing status (1 = syncing, 0 = synced)",
	})

	BatchSubmitterLastBlockHeight = factory.NewGauge(prometheus.GaugeOpts{
		Name: "opinit_batch_submitter_last_block_height",
		Help: "Batch submitter node last block height",
	})

	BatchSubmitterPendingTxs = factory.NewGauge(prometheus.GaugeOpts{
		Name: "opinit_batch_submitter_pending_txs",
		Help: "Batch submitter node pending transactions",
	})

	BatchSubmitterCurrentBatchSize = factory.NewGauge(prometheus.GaugeOpts{
		Name: "opinit_batch_submitter_current_batch_size_bytes",
		Help: "Current batch size in bytes",
	})

	BatchSubmitterBatchStartBlock = factory.NewGauge(prometheus.GaugeOpts{
		Name: "opinit_batch_submitter_batch_start_block",
		Help: "Batch start block number",
	})

	BatchSubmitterBatchEndBlock = factory.NewGauge(prometheus.GaugeOpts{
		Name: "opinit_batch_submitter_batch_end_block",
		Help: "Batch end block number",
	})

	BatchSubmitterLastSubmissionTime = factory.NewGauge(prometheus.GaugeOpts{
		Name: "opinit_batch_submitter_last_submission_time",
		Help: "Batch last submission time (unix timestamp)",
	})

	DAPendingTxs = factory.NewGauge(prometheus.GaugeOpts{
		Name: "opinit_da_pending_txs",
		Help: "DA layer pending transactions",
	})

	DAAccountSequence = factory.NewGauge(prometheus.GaugeOpts{
		Name: "opinit_da_account_sequence",
		Help: "DA account sequence number",
	})

	DALastUpdatedBatchTime = factory.NewGauge(prometheus.GaugeOpts{
		Name: "opinit_da_last_updated_batch_time",
		Help: "DA last updated batch time (unix timestamp)",
	})

	// challenger-specific metrics
	HostPendingEvents = factory.NewGaugeVec(prometheus.GaugeOpts{
		Name: "opinit_host_pending_events",
		Help: "Host pending events by event type",
	}, []string{"event_type"})

	HostPendingEventsTotal = factory.NewGauge(prometheus.GaugeOpts{
		Name: "opinit_host_pending_events_total",
		Help: "Total host pending events",
	})

	ChildPendingEvents = factory.NewGaugeVec(prometheus.GaugeOpts{
		Name: "opinit_child_pending_events",
		Help: "Child pending events by event type",
	}, []string{"event_type"})

	ChildPendingEventsTotal = factory.NewGauge(prometheus.GaugeOpts{
		Name: "opinit_child_pending_events_total",
		Help: "Total child pending events",
	})

	ChallengesTotal = factory.NewGauge(prometheus.GaugeOpts{
		Name: "opinit_challenges_total",
		Help: "Total number of challenges in latest challenges list",
	})

	LastChallengeTimestamp = factory.NewGauge(prometheus.GaugeOpts{
		Name: "opinit_last_challenge_timestamp",
		Help: "Timestamp of the most recent challenge (unix timestamp)",
	})

	// executor-specific batch info metric
	BatchInfoChainType = factory.NewGaugeVec(prometheus.GaugeOpts{
		Name: "opinit_batch_info_chain_type",
		Help: "Batch chain type (1=INITIA, 2=CELESTIA) by submitter",
	}, []string{"submitter"})
)
