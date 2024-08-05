package broadcaster

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	rpccoretypes "github.com/cometbft/cometbft/rpc/core/types"

	btypes "github.com/initia-labs/opinit-bots-go/node/broadcaster/types"
)

func (b Broadcaster) GetHeight() uint64 {
	return b.lastProcessedBlockHeight + 1
}

func (b *Broadcaster) HandleBlock(block *rpccoretypes.ResultBlock, blockResult *rpccoretypes.ResultBlockResults, latestChainHeight uint64) error {

	// check pending txs first
	for _, tx := range block.Block.Txs {
		if b.lenLocalPendingTx() == 0 {
			break
		}

		// check if the first pending tx is included in the block
		if pendingTx := b.peekLocalPendingTx(); btypes.TxHash(tx) == pendingTx.TxHash {
			err := b.MarkPendingTxAsProcessed(block.Block.Height, pendingTx.TxHash, pendingTx.Sequence)
			if err != nil {
				return err
			}
		}
	}

	// check timeout of pending txs
	// @sh-cha: should we rebroadcast pending txs? or rasing monitoring alert?
	if length := b.lenLocalPendingTx(); length > 0 {
		b.logger.Debug("remaining pending txs", zap.Int64("height", block.Block.Height), zap.Int("count", length))
		pendingTxTime := time.Unix(0, b.peekLocalPendingTx().Timestamp)
		if block.Block.Time.After(pendingTxTime.Add(btypes.TX_TIMEOUT)) {
			panic(fmt.Errorf("something wrong, pending txs are not processed for a long time; current block time: %s, pending tx processing time: %s", block.Block.Time.UTC().String(), pendingTxTime.UTC().String()))
		}
	}

	// update last processed block height
	b.lastProcessedBlockHeight = latestChainHeight

	return nil
}

func (b *Broadcaster) MarkPendingTxAsProcessed(blockHeight int64, txHash string, sequence uint64) error {
	err := b.deletePendingTx(sequence)
	if err != nil {
		return err
	}

	b.logger.Debug("tx inserted", zap.Int64("height", blockHeight), zap.Uint64("sequence", sequence), zap.String("txHash", txHash))
	b.dequeueLocalPendingTx()

	return nil
}

// CheckPendingTx checks if the pending tx is included in the block
func (b *Broadcaster) CheckPendingTx() (*btypes.PendingTxInfo, *rpccoretypes.ResultTx, error) {
	if b.lenLocalPendingTx() == 0 {
		return nil, nil, nil
	}

	pendingTx := b.peekLocalPendingTx()
	pendingTxTime := time.Unix(0, b.peekLocalPendingTx().Timestamp)
	if time.Now().After(pendingTxTime.Add(btypes.TX_TIMEOUT)) {
		// @sh-cha: should we rebroadcast pending txs? or rasing monitoring alert?
		panic(fmt.Errorf("something wrong, pending txs are not processed for a long time; current block time: %s, pending tx processing time: %s", time.Now().UTC().String(), pendingTxTime.UTC().String()))
	}

	txHash, err := hex.DecodeString(pendingTx.TxHash)
	if err != nil {
		return nil, nil, err
	}
	res, err := b.rpcClient.QueryTx(txHash)
	if err != nil {
		b.logger.Debug("failed to query tx", zap.String("txHash", pendingTx.TxHash), zap.String("error", err.Error()))
		return nil, nil, nil
	}

	return &pendingTx, res, nil
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
				select {
				case <-ctx.Done():
					return nil
				default:
				}
				err = b.handleProcessedMsgs(ctx, data)
				if err == nil {
					break
				} else if err = b.handleMsgError(err); err == nil {
					break
				}
				b.logger.Warn("retry", zap.Int("count", retry), zap.String("error", err.Error()))

				time.Sleep(30 * time.Second)
			}
			if err != nil {
				return errors.Wrap(err, "failed to handle processed msgs")
			}
		}
	}
}

// @dev: these pending processed data is filled at initialization(`NewBroadcaster`).
func (b Broadcaster) BroadcastPendingProcessedMsgs() error {
	for _, processedMsg := range b.pendingProcessedMsgs {
		b.BroadcastMsgs(processedMsg)
	}

	return nil
}
