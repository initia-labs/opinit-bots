package child

import (
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	childprovider "github.com/initia-labs/opinit-bots/provider/child"
	"github.com/initia-labs/opinit-bots/types"
	"go.uber.org/zap"
)

func (ch *Child) finalizeDepositHandler(ctx types.Context, args nodetypes.EventHandlerArgs) error {
	l1BlockHeight, l1Sequence, from, to, baseDenom, amount, err := childprovider.ParseFinalizeDeposit(args.EventAttributes)
	if err != nil {
		return err
	}
	ch.lastFinalizedDepositL1BlockHeight = l1BlockHeight
	ch.lastFinalizedDepositL1Sequence = l1Sequence

	ctx.Logger().Info("finalize token deposit",
		zap.Int64("l1_blockHeight", l1BlockHeight),
		zap.Uint64("l1_sequence", l1Sequence),
		zap.String("from", from),
		zap.String("to", to),
		zap.String("amount", amount.String()),
		zap.String("base_denom", baseDenom),
	)
	return nil
}
