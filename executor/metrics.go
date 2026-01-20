package executor

import (
	"github.com/initia-labs/opinit-bots/server/metrics"
)

// UpdateMetrics implements the metrics.Updater interface
func (ex *Executor) UpdateMetrics() error {
	status, err := ex.GetStatus()
	if err != nil {
		return err
	}

	metrics.BridgeID.Set(float64(status.BridgeId))

	// host metrics
	if status.Host.Node.Syncing != nil {
		if *status.Host.Node.Syncing {
			metrics.HostSyncing.Set(1)
		} else {
			metrics.HostSyncing.Set(0)
		}
	}

	if status.Host.Node.LastBlockHeight != nil {
		metrics.HostLastBlockHeight.Set(float64(*status.Host.Node.LastBlockHeight))
	}

	if status.Host.Node.Broadcaster != nil {
		metrics.HostPendingTxs.Set(float64(status.Host.Node.Broadcaster.PendingTxs))
		if len(status.Host.Node.Broadcaster.AccountsStatus) > 0 {
			metrics.HostAccountSequence.Set(float64(status.Host.Node.Broadcaster.AccountsStatus[0].Sequence))
		}
	}

	metrics.HostLastProposedOutputIndex.Set(float64(status.Host.LastProposedOutputIndex))
	metrics.HostLastProposedOutputL2Block.Set(float64(status.Host.LastProposedOutputL2BlockNumber))
	if !status.Host.LastProposedOutputTime.IsZero() {
		metrics.HostLastProposedOutputTime.Set(float64(status.Host.LastProposedOutputTime.Unix()))
	}

	// child metrics
	if status.Child.Node.Syncing != nil {
		if *status.Child.Node.Syncing {
			metrics.ChildSyncing.Set(1)
		} else {
			metrics.ChildSyncing.Set(0)
		}
	}

	if status.Child.Node.LastBlockHeight != nil {
		metrics.ChildLastBlockHeight.Set(float64(*status.Child.Node.LastBlockHeight))
	}

	if status.Child.Node.Broadcaster != nil {
		metrics.ChildPendingTxs.Set(float64(status.Child.Node.Broadcaster.PendingTxs))
	}

	metrics.ChildLastUpdatedOracleHeight.Set(float64(status.Child.LastUpdatedOracleL1Height))
	metrics.ChildLastFinalizedDepositL1BlockHeight.Set(float64(status.Child.LastFinalizedDepositL1BlockHeight))
	metrics.ChildLastFinalizedDepositL1Sequence.Set(float64(status.Child.LastFinalizedDepositL1Sequence))
	metrics.ChildLastWithdrawalL2Sequence.Set(float64(status.Child.LastWithdrawalL2Sequence))
	metrics.ChildWorkingTreeIndex.Set(float64(status.Child.WorkingTreeIndex))
	metrics.ChildFinalizingBlockHeight.Set(float64(status.Child.FinalizingBlockHeight))

	if !status.Child.LastOutputSubmissionTime.IsZero() {
		metrics.ChildLastOutputSubmissionTime.Set(float64(status.Child.LastOutputSubmissionTime.Unix()))
	}

	if !status.Child.NextOutputSubmissionTime.IsZero() {
		metrics.ChildNextOutputSubmissionTime.Set(float64(status.Child.NextOutputSubmissionTime.Unix()))
	}

	// batch submitter metrics
	if status.BatchSubmitter.Node.Syncing != nil {
		if *status.BatchSubmitter.Node.Syncing {
			metrics.BatchSubmitterSyncing.Set(1)
		} else {
			metrics.BatchSubmitterSyncing.Set(0)
		}
	}

	if status.BatchSubmitter.Node.LastBlockHeight != nil {
		metrics.BatchSubmitterLastBlockHeight.Set(float64(*status.BatchSubmitter.Node.LastBlockHeight))
	}

	if status.BatchSubmitter.Node.Broadcaster != nil {
		metrics.BatchSubmitterPendingTxs.Set(float64(status.BatchSubmitter.Node.Broadcaster.PendingTxs))
	}

	metrics.BatchSubmitterCurrentBatchSize.Set(float64(status.BatchSubmitter.CurrentBatchSize))
	metrics.BatchSubmitterBatchStartBlock.Set(float64(status.BatchSubmitter.BatchStartBlockNumber))
	metrics.BatchSubmitterBatchEndBlock.Set(float64(status.BatchSubmitter.BatchEndBlockNumber))

	if !status.BatchSubmitter.LastBatchSubmissionTime.IsZero() {
		metrics.BatchSubmitterLastSubmissionTime.Set(float64(status.BatchSubmitter.LastBatchSubmissionTime.Unix()))
	}

	// da metrics
	if status.DA.Node.Broadcaster != nil {
		metrics.DAPendingTxs.Set(float64(status.DA.Node.Broadcaster.PendingTxs))
		if len(status.DA.Node.Broadcaster.AccountsStatus) > 0 {
			metrics.DAAccountSequence.Set(float64(status.DA.Node.Broadcaster.AccountsStatus[0].Sequence))
		}
	}

	if !status.DA.LastUpdatedBatchTime.IsZero() {
		metrics.DALastUpdatedBatchTime.Set(float64(status.DA.LastUpdatedBatchTime.Unix()))
	}

	// batch info
	metrics.BatchInfoChainType.Reset()
	metrics.BatchInfoChainType.WithLabelValues(status.BatchSubmitter.BatchInfo.Submitter).Set(float64(status.BatchSubmitter.BatchInfo.ChainType))

	return nil
}

// MetricsUpdateInterval returns the metrics update interval in seconds, defaulting to 10 if not set
func (ex Executor) MetricsUpdateInterval() int64 {
	if ex.cfg.Server.MetricsUpdateInterval <= 0 {
		return 10
	}

	return ex.cfg.Server.MetricsUpdateInterval
}
