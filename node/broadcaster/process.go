package broadcaster

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	rpccoretypes "github.com/cometbft/cometbft/rpc/core/types"

	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
)

func (b Broadcaster) GetHeight() int64 {
	return b.lastProcessedBlockHeight + 1
}

// HandleNewBlock is called when a new block is received.
func (b *Broadcaster) HandleNewBlock(block *rpccoretypes.ResultBlock, blockResult *rpccoretypes.ResultBlockResults, latestChainHeight int64) error {
	// check pending txs first
	for _, tx := range block.Block.Txs {
		if b.LenLocalPendingTx() == 0 {
			break
		}

		// check if the first pending tx is included in the block
		if pendingTx := b.peekLocalPendingTx(); btypes.TxHash(tx) == pendingTx.TxHash {
			err := b.RemovePendingTx(block.Block.Height, pendingTx.TxHash, pendingTx.Sequence, pendingTx.MsgTypes)
			if err != nil {
				return err
			}
		}
	}

	// check timeout of pending txs
	// @sh-cha: should we rebroadcast pending txs? or rasing monitoring alert?
	if length := b.LenLocalPendingTx(); length > 0 {
		b.logger.Debug("remaining pending txs", zap.Int64("height", block.Block.Height), zap.Int("count", length))
		pendingTxTime := time.Unix(0, b.peekLocalPendingTx().Timestamp)
		if block.Block.Time.After(pendingTxTime.Add(b.cfg.TxTimeout)) {
			panic(fmt.Errorf("something wrong, pending txs are not processed for a long time; current block time: %s, pending tx processing time: %s", block.Block.Time.UTC().String(), pendingTxTime.UTC().String()))
		}
	}

	// update last processed block height
	b.lastProcessedBlockHeight = latestChainHeight

	return nil
}

// CheckPendingTx query tx info to check if pending tx is processed.
func (b *Broadcaster) CheckPendingTx(ctx context.Context) (*btypes.PendingTxInfo, *rpccoretypes.ResultTx, time.Time, error) {
	if b.LenLocalPendingTx() == 0 {
		return nil, nil, time.Time{}, nil
	}

	pendingTx := b.peekLocalPendingTx()
	pendingTxTime := time.Unix(0, b.peekLocalPendingTx().Timestamp)

	lastBlockResult, err := b.rpcClient.Block(ctx, nil)
	if err != nil {
		return nil, nil, time.Time{}, err
	}
	if lastBlockResult.Block.Time.After(pendingTxTime.Add(b.cfg.TxTimeout)) {
		// @sh-cha: should we rebroadcast pending txs? or rasing monitoring alert?
		panic(fmt.Errorf("something wrong, pending txs are not processed for a long time; current block time: %s, pending tx processing time: %s", time.Now().UTC().String(), pendingTxTime.UTC().String()))
	}

	txHash, err := hex.DecodeString(pendingTx.TxHash)
	if err != nil {
		return nil, nil, time.Time{}, err
	}
	res, err := b.rpcClient.QueryTx(ctx, txHash)
	if err != nil {
		b.logger.Debug("failed to query tx", zap.String("tx_hash", pendingTx.TxHash), zap.String("error", err.Error()))
		return nil, nil, time.Time{}, nil
	}

	blockResult, err := b.rpcClient.Block(ctx, &res.Height)
	if err != nil {
		return nil, nil, time.Time{}, err
	}
	return &pendingTx, res, blockResult.Block.Time, nil
}

// RemovePendingTx remove pending tx from local pending txs.
// It is called when the pending tx is included in the block.
func (b *Broadcaster) RemovePendingTx(blockHeight int64, txHash string, sequence uint64, msgTypes []string) error {
	err := b.deletePendingTx(sequence)
	if err != nil {
		return err
	}

	b.logger.Info("tx inserted", zap.Int64("height", blockHeight), zap.Uint64("sequence", sequence), zap.String("tx_hash", txHash), zap.Strings("msg_types", msgTypes))
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
			for retry := 1; retry <= 10; retry++ {
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
				b.logger.Warn("retry to handle processed msgs after 30 seconds", zap.Int("count", retry), zap.String("error", err.Error()))
				timer := time.NewTimer(30 * time.Second)
				select {
				case <-ctx.Done():
					return nil
				case <-timer.C:
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
