package child

import (
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	childprovider "github.com/initia-labs/opinit-bots/provider/child"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func (ch *Child) updateOracleHandler(ctx types.Context, args nodetypes.EventHandlerArgs) error {
	l1BlockHeight, from, err := childprovider.ParseUpdateOracle(args.EventAttributes)
	if err != nil {
		return errors.Wrap(err, "failed to parse update oracle event")
	}

	ch.lastUpdatedOracleL1Height = l1BlockHeight
	ctx.Logger().Info("update oracle",
		zap.Int64("l1_blockHeight", l1BlockHeight),
		zap.String("from", from),
	)
	return nil
}
