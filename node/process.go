package node

import (
	"fmt"
	"time"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	rpccoretypes "github.com/cometbft/cometbft/rpc/core/types"
	comettypes "github.com/cometbft/cometbft/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// blockProcessLooper fetches new blocks and processes them
func (n *Node) blockProcessLooper(ctx types.Context, processType nodetypes.BlockProcessType) error {
	consecutiveErrors := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			if types.SleepWithRetry(ctx, consecutiveErrors) {
				return nil
			}
			consecutiveErrors++
		}

		status, err := n.rpcClient.Status(ctx)
		if err != nil {
			ctx.Logger().Error("failed to get node status ", zap.String("error", err.Error()))
			continue
		}

		latestHeight := status.SyncInfo.LatestBlockHeight
		if n.syncedHeight >= latestHeight {
			ctx.Logger().Warn("already synced", zap.Int64("synced_height", n.syncedHeight), zap.Int64("latest_height", latestHeight))
			continue
		}

		err = n.processBlocks(ctx, processType, latestHeight)
		if nodetypes.HandleErrIgnoreAndTryLater(ctx, err) {
			ctx.Logger().Warn("ignore and try later", zap.String("error", err.Error()))
		} else if err != nil {
			ctx.Logger().Error("failed to process block", zap.String("error", err.Error()))
		} else {
			consecutiveErrors = 0
		}
	}
}

// processBlocks fetches new blocks and processes them
// if the process type is default, it will fetch blocks one by one and handle txs and events
// if the process type is raw, it will fetch blocks in bulk and send them to the raw block handler
func (n *Node) processBlocks(ctx types.Context, processType nodetypes.BlockProcessType, latestHeight int64) error {
	switch processType {
	case nodetypes.PROCESS_TYPE_DEFAULT:
		return n.processBlocksTypeDefault(ctx, latestHeight)
	case nodetypes.PROCESS_TYPE_RAW:
		return n.processBlocksTypeRaw(ctx, latestHeight)
	default:
		return errors.New("unknown block process type")
	}
}

// processBlocksTypeDefault fetches new blocks one by one and processes them
func (n *Node) processBlocksTypeDefault(ctx types.Context, latestHeight int64) error {
	timer := time.NewTicker(ctx.PollingInterval())
	defer timer.Stop()

	for height := n.syncedHeight + 1; height <= latestHeight; {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
		}

		block, blockResult, err := n.fetchNewBlock(ctx, height)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to fetch new block; height: %d", height))
		}

		err = n.handleNewBlock(ctx, block, blockResult, latestHeight)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to handle new block; height: %d", height))
		}

		n.UpdateSyncedHeight(height)
		height++
	}
	return nil
}

// processBlocksTypeRaw fetches new blocks in bulk and sends them to the raw block handler
func (n *Node) processBlocksTypeRaw(ctx types.Context, latestHeight int64) error {
	start := n.syncedHeight + 1
	end := n.syncedHeight + types.RAW_BLOCK_QUERY_MAX_SIZE
	if end > latestHeight {
		end = latestHeight
	}

	blockBulk, err := n.rpcClient.QueryBlockBulk(ctx, start, end)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to fetch block bulk: [%d, %d]", start, end))
	}

	for height := start; height <= end; height++ {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		err = n.handleRawBlock(ctx, height, latestHeight, blockBulk[height-start])
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to handle raw block: height: %d", height))
		}
		n.UpdateSyncedHeight(height)
	}
	return nil
}

// fetchNewBlock fetches a new block and block results given the height
func (n *Node) fetchNewBlock(ctx types.Context, height int64) (*rpccoretypes.ResultBlock, *rpccoretypes.ResultBlockResults, error) {
	ctx.Logger().Debug("fetch new block", zap.Int64("height", height))

	block, err := n.rpcClient.Block(ctx, &height)
	if err != nil {
		return nil, nil, err
	}

	blockResult, err := n.rpcClient.BlockResults(ctx, &height)
	if err != nil {
		return nil, nil, err
	}
	return block, blockResult, nil
}

// handleNewBlock handles a new block and block results given the height
// it sends txs and events to the respective registered handlers
func (n *Node) handleNewBlock(ctx types.Context, block *rpccoretypes.ResultBlock, blockResult *rpccoretypes.ResultBlockResults, latestChainHeight int64) error {
	protoBlock, err := block.Block.ToProto()
	if err != nil {
		return errors.Wrap(err, "failed to convert block to proto block")
	}

	err = n.handleBeginBlock(ctx, block.BlockID.Hash, protoBlock, latestChainHeight)
	if err != nil {
		return errors.Wrap(err, "failed to handle begin block")
	}

	err = n.handleBlockTxs(ctx, block, blockResult, latestChainHeight)
	if err != nil {
		return errors.Wrap(err, "failed to handle block txs")
	}

	err = n.handleFinalizeBlock(ctx, block.Block.Height, block.Block.Time.UTC(), blockResult, latestChainHeight)
	if err != nil {
		return errors.Wrap(err, "failed to handle finalize block")
	}

	err = n.handleEndBlock(ctx, block.BlockID.Hash, protoBlock, latestChainHeight)
	if err != nil {
		return errors.Wrap(err, "failed to handle end block")
	}
	return nil
}

// handleEvent handles the event for the given transaction
func (n *Node) handleEvent(ctx types.Context, blockHeight int64, blockTime time.Time, latestHeight int64, tx comettypes.Tx, txIndex int64, event abcitypes.Event) error {
	// ignore if no event handlers
	if n.eventHandlers[event.GetType()] == nil {
		return nil
	}

	ctx.Logger().Debug("handle event", zap.Int64("height", blockHeight), zap.String("type", event.GetType()))
	return n.eventHandlers[event.Type](ctx, nodetypes.EventHandlerArgs{
		BlockHeight:     blockHeight,
		BlockTime:       blockTime,
		LatestHeight:    latestHeight,
		TxIndex:         txIndex,
		Tx:              tx,
		EventAttributes: event.GetAttributes(),
	})
}
