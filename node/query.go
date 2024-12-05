package node

import (
	"context"
	"time"
)

// QueryBlockTime queries the block time of the block at the given height.
func (n Node) QueryBlockTime(ctx context.Context, height int64) (time.Time, error) {
	block, err := n.rpcClient.Block(ctx, &height)
	if err != nil {
		return time.Time{}, err
	}
	return block.Block.Header.Time, nil
}
