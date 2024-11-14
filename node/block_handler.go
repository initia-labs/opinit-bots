package node

import (
	"fmt"
	"time"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	prototypes "github.com/cometbft/cometbft/proto/tendermint/types"
	rpccoretypes "github.com/cometbft/cometbft/rpc/core/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"
)

func (n *Node) checkPendingTxsFromBroadcaster(ctx types.Context, block *rpccoretypes.ResultBlock, latestHeight int64) error {
	if n.HasBroadcaster() {
		return n.broadcaster.HandleNewBlock(ctx, block, latestHeight)
	}
	return nil
}

func (n *Node) handleBeginBlock(ctx types.Context, blockID []byte, protoBlock *prototypes.Block, latestHeight int64) error {
	if n.beginBlockHandler != nil {
		return n.beginBlockHandler(ctx, nodetypes.BeginBlockArgs{
			BlockID:      blockID,
			Block:        *protoBlock,
			LatestHeight: latestHeight,
		})
	}
	return nil
}

func (n *Node) handleBlockTxs(ctx types.Context, block *rpccoretypes.ResultBlock, blockResult *rpccoretypes.ResultBlockResults, latestHeight int64) error {
	for txIndex, tx := range block.Block.Txs {
		if n.txHandler != nil {
			err := n.txHandler(ctx, nodetypes.TxHandlerArgs{
				BlockHeight:  block.Block.Height,
				BlockTime:    block.Block.Time,
				LatestHeight: latestHeight,
				TxIndex:      int64(txIndex),
				Tx:           tx,
				Success:      blockResult.TxsResults[txIndex].Code == abcitypes.CodeTypeOK,
			})
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("failed to handle tx: tx_index: %d", txIndex))
			}
		}

		err := n.handleEvents(ctx, block.Block.Height, block.Block.Time, blockResult.TxsResults[txIndex].GetEvents(), latestHeight)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to handle events: tx_index: %d", txIndex))
		}
	}
	return nil
}

func (n *Node) handleFinalizeBlock(ctx types.Context, blockHeight int64, blockTime time.Time, blockResult *rpccoretypes.ResultBlockResults, latestHeight int64) error {
	return n.handleEvents(ctx, blockHeight, blockTime, blockResult.FinalizeBlockEvents, latestHeight)
}

func (n *Node) handleEvents(ctx types.Context, blockHeight int64, blockTime time.Time, events []abcitypes.Event, latestHeight int64) error {
	if len(n.eventHandlers) != 0 {
		for eventIndex, event := range events {
			err := n.handleEvent(ctx, blockHeight, blockTime, latestHeight, event)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("failed to handle event: event_index: %d", eventIndex))
			}
		}
	}
	return nil
}

func (n *Node) handleEndBlock(ctx types.Context, blockID []byte, protoBlock *prototypes.Block, latestHeight int64) error {
	if n.endBlockHandler != nil {
		return n.endBlockHandler(ctx, nodetypes.EndBlockArgs{
			BlockID:      blockID,
			Block:        *protoBlock,
			LatestHeight: latestHeight,
		})
	}
	return nil
}

func (n *Node) handleRawBlock(ctx types.Context, blockHeight int64, latestHeight int64, blockBytes []byte) error {
	if n.rawBlockHandler != nil {
		return n.rawBlockHandler(ctx, nodetypes.RawBlockArgs{
			BlockHeight:  blockHeight,
			LatestHeight: latestHeight,
			BlockBytes:   blockBytes,
		})
	}
	return nil
}