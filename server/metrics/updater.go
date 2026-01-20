package metrics

import (
	"time"

	"go.uber.org/zap"

	"github.com/initia-labs/opinit-bots/types"
)

// Updater interface for bots that can update their metrics
type Updater interface {
	MetricsUpdateInterval() int64
	UpdateMetrics() error
}

// StartMetricsUpdater starts periodic metrics updates for a bot that implements Updater
func StartMetricsUpdater(ctx types.Context, bot Updater) error {
	ticker := time.NewTicker(time.Duration(bot.MetricsUpdateInterval()) * time.Second)
	defer ticker.Stop()
	defer func() {
		ctx.Logger().Info("metrics updater stopped")
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := bot.UpdateMetrics(); err != nil {
				ctx.Logger().Debug("failed to update metrics", zap.Error(err))
			}
		}
	}
}
