package child

import (
	"context"
	"strconv"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"

	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
	"go.uber.org/zap"
)

func (ch *Child) updateOracleHandler(_ context.Context, args nodetypes.EventHandlerArgs) error {
	var l1BlockHeight uint64
	var from string
	var err error

	for _, attr := range args.EventAttributes {
		switch attr.Key {
		case opchildtypes.AttributeKeyHeight:
			l1BlockHeight, err = strconv.ParseUint(attr.Value, 10, 64)
			if err != nil {
				return err
			}
		case opchildtypes.AttributeKeyFrom:
			from = attr.Value
		}
	}

	ch.handleUpdateOracle(l1BlockHeight, from)
	ch.lastUpdatedOracleL1Height = l1BlockHeight
	return nil
}

func (ch *Child) handleUpdateOracle(l1BlockHeight uint64, from string) {
	ch.logger.Info("update oracle",
		zap.Uint64("l1_blockHeight", l1BlockHeight),
		zap.String("from", from),
	)
}
