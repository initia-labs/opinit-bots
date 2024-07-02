package executor

import (
	"errors"
	"fmt"
	"strconv"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	"github.com/initia-labs/opinit-bots-go/node"
)

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
	return ex.handleFinalizeDeposit(l1BlockHeight, l1Sequence, from, to, amount)
}

func (ex Executor) handleFinalizeDeposit(l1BlockHeight uint64, l1Sequence uint64, from string, to string, amount sdk.Coin) error {
	// just test deposit
	fmt.Println("Finalized deposit")
	fmt.Println("l1BlockHeight: ", l1BlockHeight)
	fmt.Println("l1Sequence: ", l1Sequence)
	fmt.Println("from: ", from)
	fmt.Println("to: ", to)
	fmt.Println("amount: ", amount)
	return nil
}
