package child

import (
	"fmt"
	"strconv"

	"cosmossdk.io/math"
	abcitypes "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
)

func ParseFinalizeDeposit(eventAttrs []abcitypes.EventAttribute) (
	l1BlockHeight, l1Sequence uint64,
	from, to, baseDenom string,
	amount sdk.Coin,
	err error) {
	for _, attr := range eventAttrs {
		switch attr.Key {
		case opchildtypes.AttributeKeyL1Sequence:
			l1Sequence, err = strconv.ParseUint(attr.Value, 10, 64)
			if err != nil {
				return
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
				err = fmt.Errorf("invalid amount %s", attr.Value)
				return
			}
			amount.Amount = coinAmount
		case opchildtypes.AttributeKeyFinalizeHeight:
			l1BlockHeight, err = strconv.ParseUint(attr.Value, 10, 64)
			if err != nil {
				return
			}
		}
	}
	return
}

func ParseUpdateOracle(eventAttrs []abcitypes.EventAttribute) (
	l1BlockHeight uint64,
	from string,
	err error) {
	for _, attr := range eventAttrs {
		switch attr.Key {
		case opchildtypes.AttributeKeyHeight:
			l1BlockHeight, err = strconv.ParseUint(attr.Value, 10, 64)
			if err != nil {
				return
			}
		case opchildtypes.AttributeKeyFrom:
			from = attr.Value
		}
	}
	return
}

func ParseInitiateWithdrawal(eventAttrs []abcitypes.EventAttribute) (
	l2Sequence, amount uint64,
	from, to, baseDenom string,
	err error) {
	for _, attr := range eventAttrs {
		switch attr.Key {
		case opchildtypes.AttributeKeyL2Sequence:
			l2Sequence, err = strconv.ParseUint(attr.Value, 10, 64)
			if err != nil {
				return
			}
		case opchildtypes.AttributeKeyFrom:
			from = attr.Value
		case opchildtypes.AttributeKeyTo:
			to = attr.Value
		case opchildtypes.AttributeKeyBaseDenom:
			baseDenom = attr.Value
		case opchildtypes.AttributeKeyAmount:
			coinAmount, ok := math.NewIntFromString(attr.Value)
			if !ok {
				err = fmt.Errorf("invalid amount %s", attr.Value)
				return
			}
			amount = coinAmount.Uint64()
		}
	}
	return
}
