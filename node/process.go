package node

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	rpccoretypes "github.com/cometbft/cometbft/rpc/core/types"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
	"go.uber.org/zap"
)

func (n *Node) blockProcessLooper(ctx context.Context, processType nodetypes.BlockProcessType) error {
	timer := time.NewTicker(nodetypes.POLLING_INTERVAL)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
		}

		status, err := n.Status(ctx)
		if err != nil {
			n.logger.Error("failed to get node status ", zap.String("error", err.Error()))
			continue
		}

		latestChainHeight := uint64(status.SyncInfo.LatestBlockHeight)
		if n.lastProcessedBlockHeight >= latestChainHeight {
			continue
		}

		switch processType {
		case nodetypes.PROCESS_TYPE_DEFAULT:
			for queryHeight := n.lastProcessedBlockHeight + 1; queryHeight <= latestChainHeight; {
				select {
				case <-ctx.Done():
					return nil
				default:
				}
				// TODO: may fetch blocks in batch
				block, blockResult, err := n.fetchNewBlock(ctx, int64(queryHeight))
				if err != nil {
					// TODO: handle error
					n.logger.Error("failed to fetch new block", zap.String("error", err.Error()))
					break
				}

				err = n.handleNewBlock(block, blockResult, latestChainHeight)
				if err != nil {
					// TODO: handle error
					n.logger.Error("failed to handle new block", zap.String("error", err.Error()))
					break
				}
				n.lastProcessedBlockHeight = queryHeight
				queryHeight++
			}

		case nodetypes.PROCESS_TYPE_RAW:
			start := n.lastProcessedBlockHeight + 1
			end := n.lastProcessedBlockHeight + 100
			if end > latestChainHeight {
				end = latestChainHeight
			}

			blockBulk, err := n.QueryBlockBulk(start, end)
			if err != nil {
				n.logger.Error("failed to fetch block bulk", zap.String("error", err.Error()))
				continue
			}

			for i := start; i <= end; i++ {
				select {
				case <-ctx.Done():
					return nil
				default:
				}
				err := n.rawBlockHandler(nodetypes.RawBlockArgs{
					BlockHeight: i,
					BlockBytes:  blockBulk[i-start],
				})
				if err != nil {
					n.logger.Error("failed to handle raw block", zap.String("error", err.Error()))
					break
				}
				n.lastProcessedBlockHeight = i
			}
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

func (n *Node) handleNewBlock(block *rpccoretypes.ResultBlock, blockResult *rpccoretypes.ResultBlockResults, latestChainHeight uint64) error {
	protoBlock, err := block.Block.ToProto()
	if err != nil {
		return err
	}
	// check pending txs first
	// TODO: may handle pending txs with same level of other handlers
	for _, tx := range block.Block.Txs {
		if n.lenLocalPendingTx() == 0 {
			break
		}

		// check if the first pending tx is included in the block
		if pendingTx := n.peekLocalPendingTx(); TxHash(tx) == pendingTx.TxHash {
			n.logger.Debug("tx inserted", zap.Int64("height", block.Block.Height), zap.Uint64("sequence", pendingTx.Sequence), zap.String("txHash", pendingTx.TxHash))
			err := n.deletePendingTx(pendingTx.Sequence)
			if err != nil {
				return err
			}
			n.dequeueLocalPendingTx()
		}
	}

	if length := n.lenLocalPendingTx(); length > 0 {
		n.logger.Debug("remaining pending txs", zap.Int64("height", block.Block.Height), zap.Int("count", length))
		pendingTxTime := time.Unix(0, n.peekLocalPendingTx().Timestamp)
		if block.Block.Time.After(pendingTxTime.Add(nodetypes.TX_TIMEOUT)) {
			// @sh-cha: should we rebroadcast pending txs? or rasing monitoring alert?
			panic(fmt.Errorf("something wrong, pending txs are not processed for a long time; current block time: %s, pending tx processing time: %s", block.Block.Time.String(), pendingTxTime.String()))
		}
	}

	if n.beginBlockHandler != nil {
		err := n.beginBlockHandler(nodetypes.BeginBlockArgs{
			BlockID:      block.BlockID.Hash,
			Block:        *protoBlock,
			LatestHeight: latestChainHeight,
		})
		if err != nil {
			return err
		}
	}

	for txIndex, tx := range block.Block.Txs {
		if n.txHandler != nil {
			err := n.txHandler(nodetypes.TxHandlerArgs{
				BlockHeight:  uint64(block.Block.Height),
				LatestHeight: latestChainHeight,
				TxIndex:      uint64(txIndex),
				Tx:           tx,
			})
			if err != nil {
				return fmt.Errorf("failed to handle tx: tx_index: %d; %w", txIndex, err)
			}
		}

		if len(n.eventHandlers) != 0 {
			events := blockResult.TxsResults[txIndex].GetEvents()
			for eventIndex, event := range events {
				err := n.handleEvent(uint64(block.Block.Height), latestChainHeight, event)
				if err != nil {
					return fmt.Errorf("failed to handle event: tx_index: %d, event_index: %d; %w", txIndex, eventIndex, err)
				}
			}
		}
	}

	for eventIndex, event := range blockResult.FinalizeBlockEvents {
		err := n.handleEvent(uint64(block.Block.Height), latestChainHeight, event)
		if err != nil {
			return fmt.Errorf("failed to handle event: finalize block, event_index: %d; %w", eventIndex, err)
		}
	}

	if n.endBlockHandler != nil {
		err := n.endBlockHandler(nodetypes.EndBlockArgs{
			BlockID:      block.BlockID.Hash,
			Block:        *protoBlock,
			LatestHeight: latestChainHeight,
		})
		if err != nil {
			return fmt.Errorf("failed to handle end block; %w", err)
		}
	}
	return nil
}

func (n *Node) handleEvent(blockHeight uint64, latestHeight uint64, event abcitypes.Event) error {
	if n.eventHandlers[event.GetType()] == nil {
		return nil
	}

	n.logger.Debug("handle event", zap.Uint64("height", blockHeight), zap.String("type", event.GetType()))
	return n.eventHandlers[event.Type](nodetypes.EventHandlerArgs{
		BlockHeight:     blockHeight,
		LatestHeight:    latestHeight,
		EventAttributes: event.GetAttributes(),
	})
}

func (n *Node) txChecker(ctx context.Context) error {
	timer := time.NewTicker(nodetypes.POLLING_INTERVAL)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
		}

		if n.lenLocalPendingTx() > 0 {
			pendingTx := n.peekLocalPendingTx()
			txHash, err := hex.DecodeString(pendingTx.TxHash)
			if err != nil {
				return err
			}
			res, err := n.QueryTx(txHash)
			if err != nil {
				n.logger.Debug("failed to query tx", zap.String("txHash", pendingTx.TxHash), zap.String("error", err.Error()))
				continue
			}
			err = n.deletePendingTx(pendingTx.Sequence)
			if err != nil {
				return err
			}
			n.logger.Debug("tx inserted", zap.Int64("height", res.Height), zap.Uint64("sequence", pendingTx.Sequence), zap.String("txHash", pendingTx.TxHash))
			n.dequeueLocalPendingTx()
		}
	}
}
