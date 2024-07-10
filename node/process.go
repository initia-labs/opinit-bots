package node

import (
	"context"
	"fmt"
	"time"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	rpccoretypes "github.com/cometbft/cometbft/rpc/core/types"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
	"go.uber.org/zap"
)

func (n *Node) blockProcessLooper(ctx context.Context) error {
	timer := time.NewTicker(nodetypes.POLLING_INTERVAL)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
		}

		status, err := n.Status(ctx)
		if err != nil {
			n.logger.Error("failed to get node status ", zap.Error(err))
			continue
		}

		latestChainHeight := status.SyncInfo.LatestBlockHeight
		if n.lastProcessedBlockHeight >= latestChainHeight {
			continue
		}

		// TODO: may fetch blocks in batch
		for queryHeight := n.lastProcessedBlockHeight + 1; queryHeight <= latestChainHeight; queryHeight++ {
			block, blockResult, err := n.fetchNewBlock(ctx, queryHeight)
			if err != nil {
				// TODO: handle error
				n.logger.Error("failed to fetch new block", zap.Error(err))
				break
			}

			err = n.handleNewBlock(block, blockResult, latestChainHeight)
			if err != nil {
				// TODO: handle error
				n.logger.Error("failed to handle new block", zap.Error(err))
				break
			}
			n.lastProcessedBlockHeight = queryHeight
		}
	}
}

func (n *Node) fetchNewBlock(ctx context.Context, height int64) (block *rpccoretypes.ResultBlock, blockResult *rpccoretypes.ResultBlockResults, err error) {
	n.logger.Debug("fetch new block", zap.Int64("height", height))
	block, err = n.Block(ctx, &height)
	if err != nil {
		return nil, nil, err
	}

	if len(n.eventHandlers) != 0 {
		blockResult, err = n.BlockResults(ctx, &height)
		if err != nil {
			return nil, nil, err
		}
	}
	return block, blockResult, nil
}

func (n *Node) handleNewBlock(block *rpccoretypes.ResultBlock, blockResult *rpccoretypes.ResultBlockResults, latestChainHeight int64) error {
	// check pending txs first
	// TODO: may handle pending txs with same level of other handlers
	for _, tx := range block.Block.Txs {
		if n.localPendingTxLength() == 0 {
			break
		} else if pendingTx := n.getLocalPendingTx(); TxHash(tx) == pendingTx.TxHash {
			n.logger.Debug("tx inserted", zap.Int64("height", block.Block.Height), zap.Uint64("sequence", pendingTx.Sequence), zap.String("txHash", pendingTx.TxHash))
			err := n.deletePendingTx(pendingTx.Sequence)
			if err != nil {
				return err
			}
			n.deleteLocalPendingTx()
		}
	}

	if length := n.localPendingTxLength(); length > 0 {
		n.logger.Debug("remaining pending txs", zap.Int64("height", block.Block.Height), zap.Int("count", length))
		pendingTxHeight := n.getLocalPendingTx().ProcessedHeight
		if block.Block.Height-pendingTxHeight > nodetypes.TIMEOUT_HEIGHT {
			panic(fmt.Errorf("something wrong, pending txs are not processed for a long time; current height: %d, pending tx processing height: %d", block.Block.Height, pendingTxHeight))
		}
	}

	if n.beginBlockHandler != nil {
		err := n.beginBlockHandler(nodetypes.BeginBlockArgs{
			BlockHeight:  block.Block.Height,
			LatestHeight: latestChainHeight,
		})
		if err != nil {
			return err
		}
	}

	for txIndex, tx := range block.Block.Txs {
		if n.txHandler != nil {
			err := n.txHandler(nodetypes.TxHandlerArgs{
				BlockHeight:  block.Block.Height,
				LatestHeight: latestChainHeight,
				TxIndex:      int64(txIndex),
				Tx:           tx,
			})
			if err != nil {
				// TODO: handle error
				return fmt.Errorf("failed to handle tx: tx_index: %d; %w", txIndex, err)
			}
		}

		if len(n.eventHandlers) != 0 {
			events := blockResult.TxsResults[txIndex].GetEvents()
			for eventIndex, event := range events {
				err := n.handleEvent(block.Block.Height, latestChainHeight, int64(txIndex), event)
				if err != nil {
					// TODO: handle error
					return fmt.Errorf("failed to handle event: tx_index: %d, event_index: %d; %w", txIndex, eventIndex, err)
				}
			}
		}
	}

	if n.endBlockHandler != nil {
		err := n.endBlockHandler(nodetypes.EndBlockArgs{
			BlockHeight:  block.Block.Height,
			LatestHeight: latestChainHeight,
		})
		if err != nil {
			return fmt.Errorf("failed to handle end block; %w", err)
		}
	}
	return nil
}

func (n *Node) handleEvent(blockHeight int64, latestHeight int64, txIndex int64, event abcitypes.Event) error {
	if n.eventHandlers[event.GetType()] == nil {
		return nil
	}
	n.logger.Debug("handle event", zap.String("name", n.name), zap.Int64("height", blockHeight), zap.String("type", event.GetType()))

	err := n.eventHandlers[event.Type](nodetypes.EventHandlerArgs{
		BlockHeight:     blockHeight,
		LatestHeight:    latestHeight,
		TxIndex:         txIndex,
		EventAttributes: event.GetAttributes(),
	})
	return err
}
