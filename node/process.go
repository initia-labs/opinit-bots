package node

import (
	"context"
	"fmt"
	"time"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	rpccoretypes "github.com/cometbft/cometbft/rpc/core/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// blockProcessLooper fetches new blocks and processes them
func (n *Node) blockProcessLooper(ctx context.Context, processType nodetypes.BlockProcessType) error {
	timer := time.NewTicker(types.PollingInterval(ctx))
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
		}

		status, err := n.rpcClient.Status(ctx)
		if err != nil {
			n.logger.Error("failed to get node status ", zap.String("error", err.Error()))
			continue
		}

		latestChainHeight := status.SyncInfo.LatestBlockHeight
		if n.lastProcessedBlockHeight >= latestChainHeight {
			continue
		}

		switch processType {
		case nodetypes.PROCESS_TYPE_DEFAULT:
			for queryHeight := n.lastProcessedBlockHeight + 1; queryHeight <= latestChainHeight; {
				select {
				case <-ctx.Done():
					return nil
				case <-timer.C:
				}
				// TODO: may fetch blocks in batch
				block, blockResult, err := n.fetchNewBlock(ctx, queryHeight)
				if err != nil {
					// TODO: handle error
					n.logger.Error("failed to fetch new block", zap.String("error", err.Error()))
					break
				}

				err = n.handleNewBlock(ctx, block, blockResult, latestChainHeight)
				if err != nil {
					n.logger.Error("failed to handle new block", zap.String("error", err.Error()))
					if errors.Is(err, nodetypes.ErrIgnoreAndTryLater) {
						sleep := time.NewTimer(time.Minute)
						select {
						case <-ctx.Done():
							return nil
						case <-sleep.C:
						}
					}
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

			blockBulk, err := n.rpcClient.QueryBlockBulk(ctx, start, end)
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
				err := n.rawBlockHandler(ctx, nodetypes.RawBlockArgs{
					BlockHeight:  i,
					LatestHeight: latestChainHeight,
					BlockBytes:   blockBulk[i-start],
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

// fetch new block from the chain
func (n *Node) fetchNewBlock(ctx context.Context, height int64) (block *rpccoretypes.ResultBlock, blockResult *rpccoretypes.ResultBlockResults, err error) {
	n.logger.Debug("fetch new block", zap.Int64("height", height))
	block, err = n.rpcClient.Block(ctx, &height)
	if err != nil {
		return nil, nil, err
	}

	if len(n.eventHandlers) != 0 {
		blockResult, err = n.rpcClient.BlockResults(ctx, &height)
		if err != nil {
			return nil, nil, err
		}
	}
	return block, blockResult, nil
}

func (n *Node) handleNewBlock(ctx context.Context, block *rpccoretypes.ResultBlock, blockResult *rpccoretypes.ResultBlockResults, latestChainHeight int64) error {
	protoBlock, err := block.Block.ToProto()
	if err != nil {
		return err
	}

	// handle broadcaster first to check pending txs
	if n.broadcaster != nil {
		err := n.broadcaster.HandleNewBlock(block, blockResult, latestChainHeight)
		if err != nil {
			return err
		}
	}

	if n.beginBlockHandler != nil {
		err := n.beginBlockHandler(ctx, nodetypes.BeginBlockArgs{
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
			err := n.txHandler(ctx, nodetypes.TxHandlerArgs{
				BlockHeight:  block.Block.Height,
				BlockTime:    block.Block.Time,
				LatestHeight: latestChainHeight,
				TxIndex:      int64(txIndex),
				Tx:           tx,
			})
			if err != nil {
				return fmt.Errorf("failed to handle tx: tx_index: %d; %w", txIndex, err)
			}
		}

		if len(n.eventHandlers) != 0 {
			events := blockResult.TxsResults[txIndex].GetEvents()
			for eventIndex, event := range events {
				err := n.handleEvent(ctx, block.Block.Height, block.Block.Time, latestChainHeight, event)
				if err != nil {
					return fmt.Errorf("failed to handle event: tx_index: %d, event_index: %d; %w", txIndex, eventIndex, err)
				}
			}
		}
	}

	if len(n.eventHandlers) != 0 {
		for eventIndex, event := range blockResult.FinalizeBlockEvents {
			err := n.handleEvent(ctx, block.Block.Height, block.Block.Time, latestChainHeight, event)
			if err != nil {
				return fmt.Errorf("failed to handle event: finalize block, event_index: %d; %w", eventIndex, err)
			}
		}
	}

	if n.endBlockHandler != nil {
		err := n.endBlockHandler(ctx, nodetypes.EndBlockArgs{
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

func (n *Node) handleEvent(ctx context.Context, blockHeight int64, blockTime time.Time, latestHeight int64, event abcitypes.Event) error {
	if n.eventHandlers[event.GetType()] == nil {
		return nil
	}

	n.logger.Debug("handle event", zap.Int64("height", blockHeight), zap.String("type", event.GetType()))
	return n.eventHandlers[event.Type](ctx, nodetypes.EventHandlerArgs{
		BlockHeight:     blockHeight,
		BlockTime:       blockTime,
		LatestHeight:    latestHeight,
		EventAttributes: event.GetAttributes(),
	})
}

// txChecker checks pending txs and handle events if the tx is included in the block
func (n *Node) txChecker(ctx context.Context) error {
	if n.broadcaster == nil {
		return nil
	}

	timer := time.NewTicker(types.PollingInterval(ctx))
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
		}

		pendingTx, res, blockTime, err := n.broadcaster.CheckPendingTx(ctx)
		if err != nil {
			return err
		} else if pendingTx == nil || res == nil {
			// tx not found
			continue
		}

		if len(n.eventHandlers) != 0 {
			events := res.TxResult.GetEvents()
			for eventIndex, event := range events {
				select {
				case <-ctx.Done():
					return nil
				default:
				}

				err := n.handleEvent(ctx, res.Height, blockTime, 0, event)
				if err != nil {
					n.logger.Error("failed to handle event", zap.String("tx_hash", pendingTx.TxHash), zap.Int("event_index", eventIndex), zap.String("error", err.Error()))
					break
				}
			}
		}

		err = n.broadcaster.RemovePendingTx(res.Height, pendingTx.TxHash, pendingTx.Sequence, pendingTx.MsgTypes)
		if err != nil {
			return err
		}
	}
}
