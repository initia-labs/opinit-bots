package types

import (
	"context"

	abci "github.com/cometbft/cometbft/abci/types"
)

type Querier interface {
	QueryABCI(ctx context.Context, req abci.RequestQuery) (abci.ResponseQuery, error)
}
