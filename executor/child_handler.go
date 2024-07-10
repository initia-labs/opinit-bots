package executor

import (
	"errors"
	"strconv"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
	"go.uber.org/zap"
)

func (ch *child) beginBlockHandler(args nodetypes.BeginBlockArgs) error {
	// just to make sure that childMsgQueue is empty
	if args.BlockHeight == args.LatestHeight && len(ch.msgQueue) != 0 && len(ch.processedMsgs) != 0 {
		panic("must not happen, msgQueue should be empty")
	}
	return nil
}

func (ch *child) endBlockHandler(args nodetypes.EndBlockArgs) error {
	// temporary 50 limit for msg queue
	// collect more msgs if block height is not latest
	if args.BlockHeight != args.LatestHeight && len(ch.msgQueue) <= 50 {
		return nil
	}

	if len(ch.msgQueue) != 0 {
		ch.processedMsgs = append(ch.processedMsgs, nodetypes.ProcessedMsgs{
			Msgs:      ch.msgQueue,
			Timestamp: time.Now().UnixNano(),
			Save:      true,
		})
	}

	// TODO: save msgs to db first with host block height sync info
	kv := ch.node.RawKVSyncInfo(args.BlockHeight)
	msgkvs, err := ch.host.RawKVProcessedData(ch.processedMsgs, false)
	if err != nil {
		return err
	}

	err = ch.db.RawBatchSet(append(msgkvs, kv)...)
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

func (ch *child) updateOracleHandler(args nodetypes.EventHandlerArgs) error {
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

func (ch *child) handleUpdateOracle(l1BlockHeight uint64, from string) {
	ch.logger.Info("update oracle",
		zap.Uint64("l1_blockHeight", l1BlockHeight),
		zap.String("from", from),
	)
}

func (ch *child) finalizeDepositHandler(args nodetypes.EventHandlerArgs) error {
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
	ch.handleFinalizeDeposit(l1BlockHeight, l1Sequence, from, to, amount)
	return nil
}

func (ch *child) handleFinalizeDeposit(l1BlockHeight uint64, l1Sequence uint64, from string, to string, amount sdk.Coin) {
	ch.logger.Info("finalize token deposit",
		zap.Uint64("l1_blockHeight", l1BlockHeight),
		zap.Uint64("l1_sequence", l1Sequence),
		zap.String("from", from),
		zap.String("to", to),
		zap.String("amount", amount.String()),
	)
}

func (ch *child) initiateWithdrawalHandler(args nodetypes.EventHandlerArgs) error {
	var l2Sequence uint64
	var from, to string
	var amount sdk.Coin
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
		case opchildtypes.AttributeKeyDenom:
			amount.Denom = attr.Value
		case opchildtypes.AttributeKeyAmount:
			coinAmount, ok := math.NewIntFromString(attr.Value)
			if !ok {
				return errors.New("invalid amount")
			}
			amount.Amount = coinAmount
		}
	}
	msg, err := ch.handleInitiateWithdrawal(l2Sequence, from, to, amount)
	if err != nil {
		return err
	}

	ch.msgQueue = append(ch.msgQueue, msg)
	return nil
}

func (ch *child) handleInitiateWithdrawal(l2Sequence uint64, from string, to string, amount sdk.Coin) (sdk.Msg, error) {

	return nil, nil
}
