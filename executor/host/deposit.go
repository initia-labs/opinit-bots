package host

import (
	"encoding/hex"
	"errors"
	"strconv"

	"cosmossdk.io/math"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (h *Host) initiateDepositHandler(args nodetypes.EventHandlerArgs) error {
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
			if bridgeId != h.bridgeId {
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

	msg, err := h.handleInitiateDeposit(
		l1Sequence,
		args.BlockHeight,
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

	h.msgQueue = append(h.msgQueue, msg)
	return nil
}

func (h *Host) handleInitiateDeposit(
	l1Sequence uint64,
	blockHeight uint64,
	from string,
	to string,
	l1Denom string,
	l2Denom string,
	amount string,
	data []byte,
) (sdk.Msg, error) {
	sender, err := h.ac.BytesToString(h.child.GetAddress())
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
	err = msg.Validate(h.ac)
	if err != nil {
		return nil, err
	}
	return msg, nil
}
