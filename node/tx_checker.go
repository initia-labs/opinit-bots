package node

import (
	"time"

	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// txChecker checks pending txs and handle events if the tx is included in the block
func (n *Node) txChecker(ctx types.Context) error {
	if !n.HasBroadcaster() {
		return nil
	}

	timer := time.NewTicker(ctx.PollingInterval())
	defer timer.Stop()

	consecutiveErrors := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
			if types.SleepWithRetry(ctx, consecutiveErrors) {
				return nil
			}
			consecutiveErrors++
		}

		pendingTx, res, blockTime, latestHeight, err := n.broadcaster.CheckPendingTx(ctx)
		if err != nil {
			ctx.Logger().Error("failed to check pending tx", zap.String("error", err.Error()))
			continue
		}

		err = n.handleEvents(ctx, res.Height, blockTime, res.TxResult.GetEvents(), latestHeight)
		if err != nil {
			ctx.Logger().Error("failed to handle events", zap.String("tx_hash", pendingTx.TxHash), zap.String("error", err.Error()))
			continue
		}

		err = n.broadcaster.RemovePendingTx(ctx, res.Height, *pendingTx)
		if err != nil {
			return errors.Wrap(err, "failed to remove pending tx")
		}
		consecutiveErrors = 0
	}
}
