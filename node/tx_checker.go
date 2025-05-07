package node

import (
	"time"

	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// txChecker continuously checks for pending transactions and handles events if the transaction is included in a block.
// If the transaction hash is not indexed by the node, even if the transaction is processed, the event handler will not be called.
// It is recommended to use the event handler only for logging or monitoring purposes.
//
// Parameters:
// - ctx: The context for managing the lifecycle of the txChecker.
// - enableEventHandler: A boolean flag to enable or disable event handling.
//
// Returns:
// - error: An error if the txChecker encounters an issue.
func (n *Node) txChecker(ctx types.Context, enableEventHandler bool) error {
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
			if n.broadcaster.LenLocalPendingTx() == 0 {
				continue
			}

			ctx.Logger().Debug("remaining pending txs", zap.Int("count", n.broadcaster.LenLocalPendingTx()))

			if types.SleepWithRetry(ctx, consecutiveErrors) {
				return nil
			}
			consecutiveErrors++
		}

		pendingTx, err := n.broadcaster.PeekLocalPendingTx()
		if err != nil {
			return err
		}

		height := int64(0)

		res, blockTime, err := n.broadcaster.CheckPendingTx(ctx, pendingTx)
		if errors.Is(err, types.ErrTxNotFound) {
			// tx not found
			// it does not check the result of the broadcast
			// this is in case the Tx gets removed from the mempool
			n.broadcaster.BroadcastTxAsync(ctx, pendingTx.Tx) //nolint:errcheck
			continue
		} else if err != nil {
			ctx.Logger().Error("failed to check pending tx", zap.String("tx_hash", pendingTx.TxHash), zap.String("error", err.Error()))
			continue
		} else if res != nil {
			// tx found
			height = res.Height
			// handle the transaction only if the node is broadcasting transactions and not processing blocks.
			if enableEventHandler && len(n.eventHandlers) != 0 {
				events := res.TxResult.GetEvents()
				for eventIndex, event := range events {
					select {
					case <-ctx.Done():
						return nil
					default:
					}

					err := n.handleEvent(ctx, res.Height, blockTime, 0, res.Tx, types.MustUint64ToInt64(uint64(res.Index)), event)
					if err != nil {
						ctx.Logger().Error("failed to handle event", zap.String("tx_hash", pendingTx.TxHash), zap.Int("event_index", eventIndex), zap.String("error", err.Error()))
						break
					}
				}
			}
		}

		err = n.broadcaster.RemovePendingTx(ctx, pendingTx)
		if err != nil {
			return errors.Wrap(err, "failed to remove pending tx")
		}
		ctx.Logger().Info("tx inserted",
			zap.Int64("height", height),
			zap.Uint64("sequence", pendingTx.Sequence),
			zap.String("tx_hash", pendingTx.TxHash),
			zap.Strings("msg_types", pendingTx.MsgTypes),
			zap.Int("pending_txs", n.broadcaster.LenLocalPendingTx()),
		)
		consecutiveErrors = 0
	}
}
