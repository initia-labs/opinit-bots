package child

import (
	"context"
	"fmt"
	"strconv"

	"cosmossdk.io/math"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"go.uber.org/zap"
)

func (ch *Child) finalizeDepositHandler(_ context.Context, args nodetypes.EventHandlerArgs) error {
	var l1BlockHeight, l1Sequence uint64
	var from, to, baseDenom string
	var amount sdk.Coin
	var err error

	for _, attr := range args.EventAttributes {
		switch attr.Key {
		case opchildtypes.AttributeKeyL1Sequence:
			l1Sequence, err = strconv.ParseUint(attr.Value, 10, 64)
			if err != nil {
				return err
			}
		case opchildtypes.AttributeKeySender:
			from = attr.Value
		case opchildtypes.AttributeKeyRecipient:
			to = attr.Value
		case opchildtypes.AttributeKeyDenom:
			amount.Denom = attr.Value
		case opchildtypes.AttributeKeyBaseDenom:
			baseDenom = attr.Value
		case opchildtypes.AttributeKeyAmount:
			coinAmount, ok := math.NewIntFromString(attr.Value)
			if !ok {
				return fmt.Errorf("invalid amount %s", attr.Value)
			}

			amount.Amount = coinAmount
		case opchildtypes.AttributeKeyFinalizeHeight:
			l1BlockHeight, err = strconv.ParseUint(attr.Value, 10, 64)
			if err != nil {
				return err
			}
		}
	}
	ch.handleFinalizeDeposit(l1BlockHeight, l1Sequence, from, to, amount, baseDenom)
	ch.lastFinalizedDepositL1BlockHeight = l1BlockHeight
	ch.lastFinalizedDepositL1Sequence = l1Sequence
	return nil
}

func (ch *Child) handleFinalizeDeposit(l1BlockHeight uint64, l1Sequence uint64, from string, to string, amount sdk.Coin, baseDenom string) {
	ch.logger.Info("finalize token deposit",
		zap.Uint64("l1_blockHeight", l1BlockHeight),
		zap.Uint64("l1_sequence", l1Sequence),
		zap.String("from", from),
		zap.String("to", to),
		zap.String("amount", amount.String()),
		zap.String("base_denom", baseDenom),
	)
}
