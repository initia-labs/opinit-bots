package child

import (
	"context"

	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	childprovider "github.com/initia-labs/opinit-bots/provider/child"
	"go.uber.org/zap"
)

func (ch *Child) updateOracleHandler(_ context.Context, args nodetypes.EventHandlerArgs) error {
	l1BlockHeight, from, err := childprovider.ParseUpdateOracle(args.EventAttributes)
	if err != nil {
		return err
	}

	ch.handleUpdateOracle(l1BlockHeight, from)
	ch.lastUpdatedOracleL1Height = l1BlockHeight
	return nil
}

func (ch *Child) handleUpdateOracle(l1BlockHeight uint64, from string) {
	ch.Logger().Info("update oracle",
		zap.Uint64("l1_blockHeight", l1BlockHeight),
		zap.String("from", from),
	)
}
