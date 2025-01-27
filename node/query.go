package node

import (
	"time"

	"github.com/initia-labs/opinit-bots/types"
)

// QueryBlockTime queries the block time of the block at the given height.
func (n Node) QueryBlockTime(ctx types.Context, height int64) (time.Time, error) {
	blockHeader, err := n.rpcClient.Header(ctx, &height)
	if err != nil {
		return time.Time{}, err
	}
	return blockHeader.Header.Time.UTC(), nil
}
