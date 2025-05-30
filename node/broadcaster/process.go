package broadcaster

import (
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	rpccoretypes "github.com/cometbft/cometbft/rpc/core/types"

	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	"github.com/initia-labs/opinit-bots/sentry_integration"
	"github.com/initia-labs/opinit-bots/types"
)

func IsTxNotFoundErr(err error, txHash string) bool {
	return strings.Contains(err.Error(), fmt.Sprintf("tx (%s) not found", txHash))
}

// CheckPendingTx query tx info to check if pending tx is processed.
func (b *Broadcaster) CheckPendingTx(ctx types.Context, pendingTx btypes.PendingTxInfo) (*rpccoretypes.ResultTx, time.Time, error) {
	txHash, err := hex.DecodeString(pendingTx.TxHash)
	if err != nil {
		return nil, time.Time{}, err
	}

	res, txerr := b.rpcClient.QueryTx(ctx, txHash)
	if txerr != nil && IsTxNotFoundErr(txerr, pendingTx.TxHash) {
		// if the tx is not found, it means the tx is not processed yet
		// or the tx is not indexed by the node in rare cases.
		pendingTxTime := time.Unix(0, pendingTx.Timestamp).UTC()

		lastHeader, err := b.rpcClient.Header(ctx, nil)
		if err != nil {
			return nil, time.Time{}, err
		}

		// before timeout
		if lastHeader.Header.Time.Before(pendingTxTime.Add(b.cfg.TxTimeout)) {
			ctx.Logger().Debug("failed to query tx", zap.String("tx_hash", pendingTx.TxHash), zap.String("error", txerr.Error()))
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
		for _, e := range ignoringErrors {
			if strings.Contains(res.TxResult.Log, e.Error()) {
				ctx.Logger().Warn("ignoring error", zap.String("error", e.Error()))
				return nil, time.Time{}, nil
			}
		}
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
func (b *Broadcaster) RemovePendingTx(ctx types.Context, pendingTx btypes.PendingTxInfo) error {
	err := DeletePendingTx(b.db, pendingTx)
	if err != nil {
		return err
	}

	b.dequeueLocalPendingTx()
	return nil
}

// Start broadcaster loop
func (b *Broadcaster) Start(ctx types.Context) error {
	defer close(b.txChannelStopped)

	for {
		select {
		case <-ctx.Done():
			return nil
		case msgs := <-b.txChannel:
			var err error
			broadcasterAccount, err := b.AccountByAddress(msgs.Sender)
			if err != nil {
				return err
			}
			for retry := 1; retry <= types.MaxRetryCount; retry++ {
				err = b.handleProcessedMsgs(ctx, msgs, broadcasterAccount)
				if err == nil {
					break
				} else if err = b.handleMsgError(ctx, err, broadcasterAccount); err == nil {
					// if the error is handled, we can delete the processed msgs
					err = DeleteProcessedMsgs(b.db, msgs)
					if err != nil {
						return err
					}
					break
				} else if !msgs.Save && !errors.Is(err, ErrAccountSequenceMismatch) {
					// if the message does not need to be saved and the error is not account sequence mismatch, we can skip retry
					ctx.Logger().Warn("discard msgs: failed to handle processed msgs", zap.String("error", err.Error()))
					err = nil
					break
				}
				ctx.Logger().Warn("retry to handle processed msgs", zap.Int("seconds", int(2*math.Exp2(float64(retry)))), zap.Int("count", retry), zap.String("error", err.Error()))
				if types.SleepWithRetry(ctx, retry) {
					return nil
				}
			}
			if err != nil {
				err = errors.Wrap(err, "failed to handle processed msgs")
				sentry_integration.CaptureCurrentHubException(err, sentry.LevelWarning)
				return err
			}
		}
	}
}

// BroadcastPendingProcessedMsgs broadcasts pending processed messages to the Broadcaster.
// It is called before the node process the block.
func (b Broadcaster) BroadcastPendingProcessedMsgs() {
	b.BroadcastProcessedMsgs(b.pendingProcessedMsgsBatch...)
}

// BroadcastProcessedMsgs broadcasts processed messages to the Broadcaster.
func (b Broadcaster) BroadcastProcessedMsgs(msgsArray ...btypes.ProcessedMsgs) {
	if b.txChannel == nil {
		return
	}

	b.broadcastProcessedMsgsMut.Lock()
	defer b.broadcastProcessedMsgsMut.Unlock()

	for _, msgs := range msgsArray {
		select {
		case <-b.txChannelStopped:
			return
		case b.txChannel <- msgs:
		}
	}
}
