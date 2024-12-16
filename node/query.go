package node

import (
	"context"
	"time"
)

// QueryBlockTime queries the block time of the block at the given height.
func (n Node) QueryBlockTime(ctx context.Context, height int64) (time.Time, error) {
	blockHeader, err := n.rpcClient.Header(ctx, &height)
	if err != nil {
		return time.Time{}, err
	}
	return blockHeader.Header.Time.UTC(), nil
}
