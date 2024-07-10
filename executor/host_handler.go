package executor

import (
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	"cosmossdk.io/math"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"

	comettypes "github.com/cometbft/cometbft/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (ex *Executor) hostBeginBlockHandler(args nodetypes.BeginBlockArgs) error {
	// just to make sure that childMsgQueue is empty
	if args.BlockHeight == args.LatestHeight && len(ex.childMsgQueue) != 0 && len(ex.hostProcessedMsgs) != 0 {
		panic("must not happen, hostMsgQueue should be empty")
	}
	return nil
}

func (ex *Executor) hostEndBlockHandler(args nodetypes.EndBlockArgs) error {
	// temporary 50 limit for msg queue
	// collect more msgs if block height is not latest
	if args.BlockHeight != args.LatestHeight && len(ex.childMsgQueue) <= 50 {
		return nil
	}

	if len(ex.childMsgQueue) != 0 {
		ex.hostProcessedMsgs = append(ex.hostProcessedMsgs, nodetypes.ProcessedMsgs{
			Msgs:      ex.childMsgQueue,
			Timestamp: time.Now().UnixNano(),
			Save:      true,
		})
	}

	// TODO: save msgs to db first with host block height sync info
	kv := ex.hostNode.RawKVSyncInfo(args.BlockHeight)
	msgkvs, err := ex.childNode.RawKVProcessedData(ex.hostProcessedMsgs, false)
	if err != nil {
		return err
	}

	err = ex.db.RawBatchSet(append(msgkvs, kv)...)
	if err != nil {
		return err
	}

	for _, processedMsg := range ex.hostProcessedMsgs {
		ex.childNode.BroadcastMsgs(processedMsg)
	}

	ex.deleteChildMsgQueue()
	ex.deleteHostProcessedMsgs()
	return nil
}

func (ex *Executor) hostTxHandler(args nodetypes.TxHandlerArgs) error {
	if args.BlockHeight == args.LatestHeight && args.TxIndex == 0 {
		msg, err := ex.oracleTxHandler(args.BlockHeight, args.Tx)
		if err != nil {
			return err
		}

		ex.hostProcessedMsgs = append(ex.hostProcessedMsgs, nodetypes.ProcessedMsgs{
			Msgs:      []sdk.Msg{msg},
			Timestamp: time.Now().UnixNano(),
			Save:      false,
		})
	}
	return nil
}

func (ex *Executor) oracleTxHandler(blockHeight int64, tx comettypes.Tx) (sdk.Msg, error) {
	sender, err := ex.ac.BytesToString(ex.childNode.GetAddress())
	if err != nil {
		return nil, err
	}

	msg := opchildtypes.NewMsgUpdateOracle(
		sender,
		uint64(blockHeight),
		tx,
	)
	err = msg.Validate(ex.ac)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (ex *Executor) initiateDepositHandler(args nodetypes.EventHandlerArgs) error {
	var bridgeId int64
	var l1Sequence uint64
	var from, to, l1Denom, l2Denom, amount string
	var data []byte
	var err error

	for _, attr := range args.EventAttributes {
		switch attr.Key {
		case ophosttypes.AttributeKeyBridgeId:
			bridgeId, err = strconv.ParseInt(attr.Value, 10, 64)
			if err != nil {
				return err
			}
			if bridgeId != ex.cfg.BridgeId {
				return errors.New("bridge ID mismatch")
			}
		case ophosttypes.AttributeKeyL1Sequence:
			l1Sequence, err = strconv.ParseUint(attr.Value, 10, 64)
			if err != nil {
				return err
			}
		case ophosttypes.AttributeKeyFrom:
			from = attr.Value
		case ophosttypes.AttributeKeyTo:
			to = attr.Value
		case ophosttypes.AttributeKeyL1Denom:
			l1Denom = attr.Value
		case ophosttypes.AttributeKeyL2Denom:
			l2Denom = attr.Value
		case ophosttypes.AttributeKeyAmount:
			amount = attr.Value
		case ophosttypes.AttributeKeyData:
			data, err = hex.DecodeString(attr.Value)
			if err != nil {
				return err
			}
		}
	}

	msg, err := ex.handleInitiateDeposit(
		l1Sequence,
		uint64(args.BlockHeight),
		from,
		to,
		l1Denom,
		l2Denom,
		amount,
		data,
	)
	if err != nil {
		return err
	}

	ex.childMsgQueue = append(ex.childMsgQueue, msg)
	return nil
}

func (ex *Executor) handleInitiateDeposit(
	l1Sequence uint64,
	blockHeight uint64,
	from string,
	to string,
	l1Denom string,
	l2Denom string,
	amount string,
	data []byte,
) (sdk.Msg, error) {
	sender, err := ex.ac.BytesToString(ex.childNode.GetAddress())
	if err != nil {
		return nil, err
	}
	coinAmount, ok := math.NewIntFromString(amount)
	if !ok {
		return nil, errors.New("invalid amount")
	}
	coin := sdk.NewCoin(l2Denom, coinAmount)

	msg := opchildtypes.NewMsgFinalizeTokenDeposit(
		sender,
		from,
		to,
		coin,
		l1Sequence,
		blockHeight,
		l1Denom,
		data,
	)
	err = msg.Validate(ex.ac)
	if err != nil {
		return nil, err
	}
	return msg, nil
}
