package broadcaster

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	rpccoretypes "github.com/cometbft/cometbft/rpc/core/types"

	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	"github.com/initia-labs/opinit-bots/types"
)

func (b Broadcaster) GetHeight() int64 {
	return b.lastProcessedBlockHeight + 1
}

// CheckPendingTx query tx info to check if pending tx is processed.
func (b *Broadcaster) CheckPendingTx(ctx context.Context, pendingTx btypes.PendingTxInfo) (*rpccoretypes.ResultTx, time.Time, error) {
	txHash, err := hex.DecodeString(pendingTx.TxHash)
	if err != nil {
		return nil, time.Time{}, err
	}
	res, txerr := b.rpcClient.QueryTx(ctx, txHash)
	if txerr != nil {
		// if the tx is not found, it means the tx is not processed yet
		// or the tx is not indexed by the node in rare cases.
		lastHeader, err := b.rpcClient.Header(ctx, nil)
		if err != nil {
			return nil, time.Time{}, err
		}
		pendingTxTime := time.Unix(0, pendingTx.Timestamp)

		// before timeout
		if lastHeader.Header.Time.Before(pendingTxTime.Add(b.cfg.TxTimeout)) {
			b.logger.Debug("failed to query tx", zap.String("tx_hash", pendingTx.TxHash), zap.String("error", txerr.Error()))
			return nil, time.Time{}, types.ErrTxNotFound
		} else {
			// timeout case
			account, err := b.GetAccount(b.getClientCtx(ctx), b.keyAddress)
			if err != nil {
				return nil, time.Time{}, err
			}

			// if sequence is larger than the sequence of the pending tx,
			// handle it as the tx has already been processed
			if pendingTx.Sequence < account.GetSequence() {
				return nil, time.Time{}, nil
			}
			panic(fmt.Errorf("something wrong, pending txs are not processed for a long time; current block time: %s, pending tx processing time: %s", time.Now().UTC().String(), pendingTxTime.UTC().String()))
		}
	} else if res.TxResult.Code != 0 {
		panic(fmt.Errorf("tx failed, tx hash: %s, code: %d, log: %s; you might need to check gas adjustment config or balance", pendingTx.TxHash, res.TxResult.Code, res.TxResult.Log))
	}

	header, err := b.rpcClient.Header(ctx, &res.Height)
	if err != nil {
		return nil, time.Time{}, err
	}
	return res, header.Header.Time, nil
}

// RemovePendingTx remove pending tx from local pending txs.
// It is called when the pending tx is included in the block.
func (b *Broadcaster) RemovePendingTx(sequence uint64) error {
	err := b.deletePendingTx(sequence)
	if err != nil {
		return err
	}

	b.dequeueLocalPendingTx()
	return nil
}

// Start broadcaster loop
func (b *Broadcaster) Start(ctx context.Context) error {
	defer close(b.txChannelStopped)

	for {
		select {
		case <-ctx.Done():
			return nil
		case data := <-b.txChannel:
			var err error
			for retry := 1; retry <= types.MaxRetryCount; retry++ {
				err = b.handleProcessedMsgs(ctx, data)
				if err == nil {
					break
				} else if err = b.handleMsgError(err); err == nil {
					break
				} else if !data.Save {
					// if the message does not need to be saved, we can skip retry
					err = nil
					break
				}
				b.logger.Warn(fmt.Sprintf("retry to handle processed msgs after %d seconds", int(2*math.Exp2(float64(retry)))), zap.Int("count", retry), zap.String("error", err.Error()))
				if types.SleepWithRetry(ctx, retry) {
					return nil
				}
			}
			if err != nil {
				return errors.Wrap(err, "failed to handle processed msgs")
			}
		}
	}
}

// @dev: these pending processed data is filled at initialization(`NewBroadcaster`).
func (b Broadcaster) BroadcastPendingProcessedMsgs() {
	for _, processedMsg := range b.pendingProcessedMsgs {
		b.BroadcastMsgs(processedMsg)
	}
}
