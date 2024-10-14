package child

import (
	"fmt"
	"strconv"

	"cosmossdk.io/math"
	abcitypes "github.com/cometbft/cometbft/abci/types"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func missingAttrsError(missingAttrs map[string]struct{}) error {
	if len(missingAttrs) != 0 {
		missingAttrStr := ""
		for attr := range missingAttrs {
			missingAttrStr += attr + " "
		}
		return fmt.Errorf("missing attributes: %s", missingAttrs)
	}
	return nil
}

func ParseFinalizeDeposit(eventAttrs []abcitypes.EventAttribute) (
	l1BlockHeight int64,
	l1Sequence uint64,
	from, to, baseDenom string,
	amount sdk.Coin,
	err error) {
	missingAttrs := map[string]struct{}{
		opchildtypes.AttributeKeyL1Sequence:     {},
		opchildtypes.AttributeKeySender:         {},
		opchildtypes.AttributeKeyRecipient:      {},
		opchildtypes.AttributeKeyDenom:          {},
		opchildtypes.AttributeKeyBaseDenom:      {},
		opchildtypes.AttributeKeyAmount:         {},
		opchildtypes.AttributeKeyFinalizeHeight: {},
	}

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
			l1BlockHeight, err = strconv.ParseInt(attr.Value, 10, 64)
			if err != nil {
				return
			}
		default:
			continue
		}
		delete(missingAttrs, attr.Key)
	}
	err = missingAttrsError(missingAttrs)
	return
}

func ParseUpdateOracle(eventAttrs []abcitypes.EventAttribute) (
	l1BlockHeight int64,
	from string,
	err error) {
	missingAttrs := map[string]struct{}{
		opchildtypes.AttributeKeyHeight: {},
		opchildtypes.AttributeKeyFrom:   {},
	}

	for _, attr := range eventAttrs {
		switch attr.Key {
		case opchildtypes.AttributeKeyHeight:
			l1BlockHeight, err = strconv.ParseInt(attr.Value, 10, 64)
			if err != nil {
				return
			}
		case opchildtypes.AttributeKeyFrom:
			from = attr.Value
		default:
			continue
		}
		delete(missingAttrs, attr.Key)
	}
	err = missingAttrsError(missingAttrs)
	return
}

func ParseInitiateWithdrawal(eventAttrs []abcitypes.EventAttribute) (
	l2Sequence, amount uint64,
	from, to, baseDenom string,
	err error) {
	missingAttrs := map[string]struct{}{
		opchildtypes.AttributeKeyL2Sequence: {},
		opchildtypes.AttributeKeyFrom:       {},
		opchildtypes.AttributeKeyTo:         {},
		opchildtypes.AttributeKeyBaseDenom:  {},
		opchildtypes.AttributeKeyAmount:     {},
	}

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
		default:
			continue
		}
		delete(missingAttrs, attr.Key)
	}
	err = missingAttrsError(missingAttrs)
	return
}
