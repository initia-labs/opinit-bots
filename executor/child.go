package executor

import (
	"errors"
	"strconv"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	"github.com/initia-labs/opinit-bots-go/node"
	"go.uber.org/zap"
)

func (ex Executor) updateOracleHandler(args node.EventHandlerArgs) error {
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
	ex.handleUpdateOracle(l1BlockHeight, from)
	return nil
}

func (ex Executor) handleUpdateOracle(l1BlockHeight uint64, from string) {
	ex.logger.Info("update oracle",
		zap.Uint64("l1_blockHeight", l1BlockHeight),
		zap.String("from", from),
	)
}

func (ex Executor) finalizeDepositHandler(args node.EventHandlerArgs) error {
	var l1BlockHeight, l1Sequence uint64
	var from, to string
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
		case opchildtypes.AttributeKeyAmount:
			coinAmount, ok := math.NewIntFromString(attr.Value)
			if !ok {
				return errors.New("invalid amount")
			}
			amount.Amount = coinAmount
		case opchildtypes.AttributeKeyFinalizeHeight:
			l1BlockHeight, err = strconv.ParseUint(attr.Value, 10, 64)
			if err != nil {
				return err
			}
		}
	}
	ex.handleFinalizeDeposit(l1BlockHeight, l1Sequence, from, to, amount)
	return nil
}

func (ex Executor) handleFinalizeDeposit(l1BlockHeight uint64, l1Sequence uint64, from string, to string, amount sdk.Coin) {
	ex.logger.Info("finalize token deposit",
		zap.Uint64("l1_blockHeight", l1BlockHeight),
		zap.Uint64("l1_sequence", l1Sequence),
		zap.String("from", from),
		zap.String("to", to),
		zap.String("amount", amount.String()),
	)
}
