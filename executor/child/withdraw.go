package child

import (
	"errors"
	"strconv"
	"time"

	"cosmossdk.io/math"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	executortypes "github.com/initia-labs/opinit-bots-go/executor/types"

	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
)

func (ch *Child) initiateWithdrawalHandler(args nodetypes.EventHandlerArgs) error {
	var l2Sequence, amount uint64
	var from, to, baseDenom string
	var err error

	for _, attr := range args.EventAttributes {
		switch attr.Key {
		case opchildtypes.AttributeKeyL2Sequence:
			l2Sequence, err = strconv.ParseUint(attr.Value, 10, 64)
			if err != nil {
				return err
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
				return errors.New("invalid amount")
			}
			amount = coinAmount.Uint64()
		}
	}
	ch.handleInitiateWithdrawal(l2Sequence, from, to, baseDenom, amount)
	return nil
}

func (ch *Child) handleInitiateWithdrawal(l2Sequence uint64, from string, to string, baseDenom string, amount uint64) {
	withdrawal := ophosttypes.GenerateWithdrawalHash(uint64(ch.bridgeId), l2Sequence, from, to, baseDenom, amount)
	ch.blockWithdrawals.Withdrawals = append(ch.blockWithdrawals.Withdrawals, executortypes.Withdrawal{
		Sequence:       l2Sequence,
		WithdrawalHash: withdrawal,
	})
}

func (ch *Child) prepareWithdrawals(blockHeight int64) error {
	ch.blockWithdrawals = executortypes.Withdrawals{
		Height: uint64(blockHeight),
	}

	if ch.nextSentOutputTime.IsZero() || time.Now().After(ch.nextSentOutputTime) {
		output, err := ch.host.QueryLastOutput()
		if err != nil {
			return err
		}
		ch.nextSentOutputTime = output.OutputProposal.L1BlockTime.Add(executortypes.OutputInterval)
		ch.lastSentOutputBlockHeight = output.OutputProposal.L2BlockNumber
	}
	return nil
}

func (ch *Child) generateOutputRoot(blockHeight int64) error {

	return nil
}
