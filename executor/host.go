package executor

import (
	"encoding/hex"
	"errors"
	"strconv"

	"cosmossdk.io/math"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/initia-labs/opinit-bots-go/node"

	comettypes "github.com/cometbft/cometbft/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (ex Executor) hostTxHandler(args node.TxHandlerArgs) error {
	if args.TxIndex == 0 {
		return ex.oracleTxHandler(args.BlockHeight, args.Tx)
	}
	return nil
}

func (ex Executor) oracleTxHandler(blockHeight int64, tx comettypes.Tx) error {
	sender, err := ex.ac.BytesToString(ex.childNode.GetAddress())
	if err != nil {
		return err
	}

	msg := opchildtypes.NewMsgUpdateOracle(
		sender,
		uint64(blockHeight),
		tx,
	)
	err = msg.Validate(ex.ac)
	if err != nil {
		return err
	}

	ex.childNode.SendTx(msg)
	return nil
}

func (ex Executor) initiateDepositHandler(args node.EventHandlerArgs) error {
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

	return ex.handleInitiateDeposit(
		l1Sequence,
		uint64(args.BlockHeight),
		from,
		to,
		l1Denom,
		l2Denom,
		amount,
		data,
	)
}

func (ex Executor) handleInitiateDeposit(
	l1Sequence uint64,
	blockHeight uint64,
	from string,
	to string,
	l1Denom string,
	l2Denom string,
	amount string,
	data []byte,
) error {
	sender, err := ex.ac.BytesToString(ex.childNode.GetAddress())
	if err != nil {
		return err
	}
	coinAmount, ok := math.NewIntFromString(amount)
	if !ok {
		return errors.New("invalid amount")
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
		return err
	}

	ex.childNode.SendTx(msg)
	return nil
}
