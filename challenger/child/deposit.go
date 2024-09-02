package child

import (
	"context"
	"time"

	childprovider "github.com/initia-labs/opinit-bots/provider/child"

	sdk "github.com/cosmos/cosmos-sdk/types"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
)

func (ch *Child) finalizeDepositHandler(_ context.Context, args nodetypes.EventHandlerArgs) error {
	l1BlockHeight, l1Sequence, from, to, baseDenom, amount, err := childprovider.ParseFinalizeDeposit(args.EventAttributes)
	if err != nil {
		return err
	}
	ch.handleFinalizeDeposit(args.BlockTime, l1BlockHeight, l1Sequence, from, to, amount, baseDenom)
	ch.lastFinalizedDepositL1BlockHeight = l1BlockHeight
	ch.lastFinalizedDepositL1Sequence = l1Sequence
	return nil
}

func (ch *Child) handleFinalizeDeposit(l2BlockTime time.Time, l1BlockHeight uint64, l1Sequence uint64, from string, to string, amount sdk.Coin, baseDenom string) {
	deposit := challengertypes.NewDeposit(l1Sequence, l1BlockHeight, from, to, baseDenom, amount.String(), l2BlockTime)
	ch.eventQueue = append(ch.eventQueue, deposit)
}
