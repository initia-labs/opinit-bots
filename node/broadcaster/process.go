package broadcaster

import (
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

// HandleNewBlock is called when a new block is received.
func (b *Broadcaster) HandleNewBlock(ctx types.Context, block *rpccoretypes.ResultBlock, latestChainHeight int64) error {
	// check pending txs first
	for _, tx := range block.Block.Txs {
		if b.LenLocalPendingTx() == 0 {
			break
		}

		// check if the first pending tx is included in the block
		if pendingTx := b.peekLocalPendingTx(); btypes.TxHash(tx) == pendingTx.TxHash {
			err := b.RemovePendingTx(ctx, block.Block.Height, pendingTx)
			if err != nil {
				return errors.Wrap(err, "failed to remove pending tx")
			}
		}
	}

	// check timeout of pending txs
	// @sh-cha: should we rebroadcast pending txs? or rasing monitoring alert?
	if length := b.LenLocalPendingTx(); length > 0 {
		ctx.Logger().Debug("remaining pending txs", zap.Int64("height", block.Block.Height), zap.Int("count", length))
		pendingTxTime := time.Unix(0, b.peekLocalPendingTx().Timestamp)
		if block.Block.Time.After(pendingTxTime.Add(b.cfg.TxTimeout)) {
			panic(fmt.Errorf("something wrong, pending txs are not processed for a long time; current block time: %s, pending tx processing time: %s", block.Block.Time.UTC().String(), pendingTxTime.UTC().String()))
		}
	}
	return nil
}

// CheckPendingTx query tx info to check if pending tx is processed.
func (b *Broadcaster) CheckPendingTx(ctx types.Context) (*btypes.PendingTxInfo, *rpccoretypes.ResultTx, time.Time, int64, error) {
	if b.LenLocalPendingTx() == 0 {
		return nil, nil, time.Time{}, 0, nil
	}

	pendingTx := b.peekLocalPendingTx()
	pendingTxTime := time.Unix(0, b.peekLocalPendingTx().Timestamp)

	latestBlockResult, err := b.rpcClient.Block(ctx, nil)
	if err != nil {
		return nil, nil, time.Time{}, 0, errors.Wrap(err, "failed to fetch latest block")
	}
	if latestBlockResult.Block.Time.After(pendingTxTime.Add(b.cfg.TxTimeout)) {
		// @sh-cha: should we rebroadcast pending txs? or rasing monitoring alert?
		panic(fmt.Errorf("something wrong, pending txs are not processed for a long time; current block time: %s, pending tx processing time: %s", time.Now().UTC().String(), pendingTxTime.UTC().String()))
	}

	txHash, err := hex.DecodeString(pendingTx.TxHash)
	if err != nil {
		return nil, nil, time.Time{}, 0, errors.Wrap(err, "failed to decode tx hash")
	}
	res, err := b.rpcClient.QueryTx(ctx, txHash)
	if err != nil {
		ctx.Logger().Debug("failed to query tx", zap.String("tx_hash", pendingTx.TxHash), zap.String("error", err.Error()))
		return nil, nil, time.Time{}, 0, errors.Wrap(err, fmt.Sprintf("failed to query tx; hash: %s", pendingTx.TxHash))
	}

	blockResult, err := b.rpcClient.Block(ctx, &res.Height)
	if err != nil {
		return nil, nil, time.Time{}, 0, errors.Wrap(err, fmt.Sprintf("failed to fetch block; height: %d", res.Height))
	}
	return &pendingTx, res, blockResult.Block.Time, latestBlockResult.Block.Height, nil
}

// RemovePendingTx remove pending tx from local pending txs.
// It is called when the pending tx is included in the block.
func (b *Broadcaster) RemovePendingTx(ctx types.Context, blockHeight int64, pendingTx btypes.PendingTxInfo) error {
	err := DeletePendingTx(b.db, pendingTx)
	if err != nil {
		return err
	}

	ctx.Logger().Info("tx inserted", zap.Int64("height", blockHeight), zap.Uint64("sequence", pendingTx.Sequence), zap.String("tx_hash", pendingTx.TxHash), zap.Strings("msg_types", pendingTx.MsgTypes))
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
			for retry := 1; retry <= types.MaxRetryCount; retry++ {
				err = b.handleProcessedMsgs(ctx, msgs)
				if err == nil {
					break
				} else if err = b.handleMsgError(ctx, err); err == nil {
					break
				} else if !msgs.Save {
					// if the message does not need to be saved, we can skip retry
					err = nil
					break
				}
				ctx.Logger().Warn("retry to handle processed msgs", zap.Int("seconds", int(2*math.Exp2(float64(retry)))), zap.Int("count", retry), zap.String("error", err.Error()))
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

// BroadcastTxSync broadcasts transaction bytes to txBroadcastLooper.
func (b Broadcaster) BroadcastProcessedMsgs(msgs btypes.ProcessedMsgs) {
	if b.txChannel == nil {
		return
	}

	select {
	case <-b.txChannelStopped:
	case b.txChannel <- msgs:
	}
}

// @dev: these pending processed data is filled at initialization(`NewBroadcaster`).
func (b Broadcaster) BroadcastPendingProcessedMsgs() {
	for _, processedMsg := range b.pendingProcessedMsgsBatch {
		b.BroadcastProcessedMsgs(processedMsg)
	}
}
