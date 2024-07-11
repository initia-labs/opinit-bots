package child

import (
	"errors"
	"strconv"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
	"github.com/initia-labs/opinit-bots-go/types"
	"go.uber.org/zap"
)

func (ch *Child) beginBlockHandler(args nodetypes.BeginBlockArgs) error {
	// just to make sure that childMsgQueue is empty
	if args.BlockHeight == args.LatestHeight && len(ch.msgQueue) != 0 && len(ch.processedMsgs) != 0 {
		panic("must not happen, msgQueue should be empty")
	}

	return nil
}

func (ch *Child) endBlockHandler(args nodetypes.EndBlockArgs) error {
	// temporary 50 limit for msg queue
	// collect more msgs if block height is not latest
	if args.BlockHeight != args.LatestHeight && len(ch.msgQueue) <= 50 {
		return nil
	}

	batchKVs := []types.KV{
		ch.node.RawKVSyncInfo(args.BlockHeight),
	}
	if ch.node.HasKey() {
		if len(ch.msgQueue) != 0 {
			ch.processedMsgs = append(ch.processedMsgs, nodetypes.ProcessedMsgs{
				Msgs:      ch.msgQueue,
				Timestamp: time.Now().UnixNano(),
				Save:      true,
			})
		}
		msgkvs, err := ch.host.RawKVProcessedData(ch.processedMsgs, false)
		if err != nil {
			return err
		}
		batchKVs = append(batchKVs, msgkvs...)
	}

	err := ch.db.RawBatchSet(batchKVs...)
	if err != nil {
		return err
	}

	for _, processedMsg := range ch.processedMsgs {
		ch.host.BroadcastMsgs(processedMsg)
	}

	ch.msgQueue = ch.msgQueue[:0]
	ch.processedMsgs = ch.processedMsgs[:0]
	return nil
}

func (ch *Child) updateOracleHandler(args nodetypes.EventHandlerArgs) error {
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
	return nil
}

func (ch *Child) handleUpdateOracle(l1BlockHeight uint64, from string) {
	ch.logger.Info("update oracle",
		zap.Uint64("l1_blockHeight", l1BlockHeight),
		zap.String("from", from),
	)
}

func (ch *Child) finalizeDepositHandler(args nodetypes.EventHandlerArgs) error {
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
	ch.handleFinalizeDeposit(l1BlockHeight, l1Sequence, from, to, amount, baseDenom)
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
	err = ch.handleInitiateWithdrawal(l2Sequence, from, to, baseDenom, amount)
	if err != nil {
		return err
	}
	return nil
}

func (ch *Child) handleInitiateWithdrawal(l2Sequence uint64, from string, to string, baseDenom string, amount uint64) error {
	withdrawal := ophosttypes.GenerateWithdrawalHash(uint64(ch.bridgeId), l2Sequence, from, to, baseDenom, amount)
	return ch.saveWithdrawal(l2Sequence, withdrawal)
}
