package node

import (
	"context"
	"time"

	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// txChecker checks pending txs and handle events if the tx is included in the block
func (n *Node) txChecker(ctx context.Context) error {
	if !n.HasBroadcaster() {
		return nil
	}

	timer := time.NewTicker(types.PollingInterval(ctx))
	defer timer.Stop()

	consecutiveErrors := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
			types.SleepWithRetry(ctx, consecutiveErrors)
			consecutiveErrors++
		}

		pendingTx, res, blockTime, latestHeight, err := n.broadcaster.CheckPendingTx(ctx)
		if err != nil {
			n.logger.Error("failed to check pending tx", zap.String("error", err.Error()))
			continue
		}

		err = n.handleEvents(ctx, res.Height, blockTime, res.TxResult.GetEvents(), latestHeight)
		if err != nil {
			n.logger.Error("failed to handle events", zap.String("tx_hash", pendingTx.TxHash), zap.String("error", err.Error()))
			continue
		}

		err = n.broadcaster.RemovePendingTx(res.Height, pendingTx.TxHash, pendingTx.Sequence, pendingTx.MsgTypes)
		if err != nil {
			return errors.Wrap(err, "failed to remove pending tx")
		}
		consecutiveErrors = 0
	}
}
