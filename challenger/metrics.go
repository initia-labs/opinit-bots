package challenger

import (
	"github.com/initia-labs/opinit-bots/server/metrics"
)

// UpdateMetrics implements the metrics.Updater interface
func (c *Challenger) UpdateMetrics() error {
	status, err := c.GetStatus()
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

	// host-specific challenger metrics (pending events)
	if status.Host.NumPendingEvents != nil {
		metrics.HostPendingEvents.Reset()
		totalPendingEvents := int64(0)
		for eventType, count := range status.Host.NumPendingEvents {
			metrics.HostPendingEvents.WithLabelValues(eventType).Set(float64(count))
			totalPendingEvents += count
		}
		metrics.HostPendingEventsTotal.Set(float64(totalPendingEvents))
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

	// child-specific challenger metrics
	if status.Child.NumPendingEvents != nil {
		metrics.ChildPendingEvents.Reset()
		totalPendingEvents := int64(0)
		for eventType, count := range status.Child.NumPendingEvents {
			metrics.ChildPendingEvents.WithLabelValues(eventType).Set(float64(count))
			totalPendingEvents += count
		}
		metrics.ChildPendingEventsTotal.Set(float64(totalPendingEvents))
	}

	// challenges metrics
	metrics.ChallengesTotal.Set(float64(len(status.LatestChallenges)))
	if len(status.LatestChallenges) > 0 {
		var latestTimestamp int64
		for _, challenge := range status.LatestChallenges {
			if challenge.Time.Unix() > latestTimestamp {
				latestTimestamp = challenge.Time.Unix()
			}
		}
		metrics.LastChallengeTimestamp.Set(float64(latestTimestamp))
	}

	return nil
}

// MetricsUpdateInterval returns the metrics update interval in seconds, defaulting to 10 if not set
func (c Challenger) MetricsUpdateInterval() int64 {
	if c.cfg.Server.MetricsUpdateInterval <= 0 {
		return 10
	}

	return c.cfg.Server.MetricsUpdateInterval
}
