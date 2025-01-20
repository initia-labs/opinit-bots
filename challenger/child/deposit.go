package child

import (
	childprovider "github.com/initia-labs/opinit-bots/provider/child"
	"github.com/initia-labs/opinit-bots/types"
	"go.uber.org/zap"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/pkg/errors"
)

func (ch *Child) finalizeDepositHandler(ctx types.Context, args nodetypes.EventHandlerArgs) error {
	l1BlockHeight, l1Sequence, from, to, baseDenom, amount, err := childprovider.ParseFinalizeDeposit(args.EventAttributes)
	if err != nil {
		return errors.Wrap(err, "failed to parse finalize deposit event")
	}
	deposit := challengertypes.NewDeposit(l1Sequence, l1BlockHeight, from, to, baseDenom, amount.String(), args.BlockTime)
	ch.eventQueue = append(ch.eventQueue, deposit)

	ctx.Logger().Info("finalize token deposit",
		zap.Int64("l1_blockHeight", l1BlockHeight),
		zap.Uint64("l1_sequence", l1Sequence),
		zap.String("from", from),
		zap.String("to", to),
		zap.String("amount", amount.String()),
		zap.String("base_denom", baseDenom),
	)

	ch.lastFinalizedDepositL1BlockHeight = l1BlockHeight
	ch.lastFinalizedDepositL1Sequence = l1Sequence
	return nil
}
