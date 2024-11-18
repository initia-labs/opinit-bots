package broadcaster

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	rpccoretypes "github.com/cometbft/cometbft/rpc/core/types"

	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	"github.com/initia-labs/opinit-bots/types"
)

func IsTxNotFoundErr(err error, txHash string) bool {
	return strings.Contains(err.Error(), fmt.Sprintf("tx (%s) not found", txHash))
}

// CheckPendingTx query tx info to check if pending tx is processed.
func (b *Broadcaster) CheckPendingTx(ctx context.Context, pendingTx btypes.PendingTxInfo) (*rpccoretypes.ResultTx, time.Time, error) {
	txHash, err := hex.DecodeString(pendingTx.TxHash)
	if err != nil {
		return nil, time.Time{}, err
	}

	res, txerr := b.rpcClient.QueryTx(ctx, txHash)
	if txerr != nil && IsTxNotFoundErr(txerr, pendingTx.TxHash) {
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
			account, err := b.AccountByAddress(pendingTx.Sender)
			if err != nil {
				return nil, time.Time{}, err
			}
			accountSequence, err := account.GetLatestSequence(ctx)
			if err != nil {
				return nil, time.Time{}, err
			}

			// if sequence is larger than the sequence of the pending tx,
			// handle it as the tx has already been processed
			if pendingTx.Sequence < accountSequence {
				return nil, time.Time{}, nil
			}
			panic(fmt.Errorf("something wrong, pending txs are not processed for a long time; current block time: %s, pending tx processing time: %s", time.Now().UTC().String(), pendingTxTime.UTC().String()))
		}
	} else if txerr != nil {
		return nil, time.Time{}, txerr
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
func (b *Broadcaster) RemovePendingTx(pendingTx btypes.PendingTxInfo) error {
	err := b.deletePendingTx(pendingTx)
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
			broadcasterAccount, err := b.AccountByAddress(data.Sender)
			if err != nil {
				return err
			}
			for retry := 1; retry <= types.MaxRetryCount; retry++ {
				err = b.handleProcessedMsgs(ctx, data, broadcasterAccount)
				if err == nil {
					break
				} else if err = b.handleMsgError(err, broadcasterAccount); err == nil {
					// if the error is handled, we can delete the processed msgs
					err = b.deleteProcessedMsgs(data.Timestamp)
					if err != nil {
						return err
					}
					break
				} else if !data.Save {
					b.logger.Warn("discard msgs: failed to handle processed msgs", zap.String("error", err.Error()))
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

// BroadcastTxSync broadcasts transaction bytes to txBroadcastLooper.
func (b Broadcaster) BroadcastMsgs(msgs btypes.ProcessedMsgs) {
	if b.txChannel == nil {
		return
	}

	select {
	case <-b.txChannelStopped:
	case b.txChannel <- msgs:
	}
}
