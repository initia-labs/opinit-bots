package host

import (
	"context"
	"encoding/hex"
	"errors"
	"strconv"

	"cosmossdk.io/math"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (h *Host) initiateDepositHandler(_ context.Context, args nodetypes.EventHandlerArgs) error {
	var bridgeId uint64
	var l1Sequence uint64
	var from, to, l1Denom, l2Denom, amount string
	var data []byte
	var err error

	for _, attr := range args.EventAttributes {
		switch attr.Key {
		case ophosttypes.AttributeKeyBridgeId:
			bridgeId, err = strconv.ParseUint(attr.Value, 10, 64)
			if err != nil {
				return err
			}
			if bridgeId != uint64(h.bridgeId) {
				// pass other bridge deposit event
				return nil
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
	if l1Sequence < h.initialL1Sequence {
		// pass old deposit event
		return nil
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
	coinAmount, ok := math.NewIntFromString(amount)
	if !ok {
		return nil, errors.New("invalid amount")
	}
	coin := sdk.NewCoin(l2Denom, coinAmount)

	return h.child.GetMsgFinalizeTokenDeposit(
		from,
		to,
		coin,
		l1Sequence,
		blockHeight,
		l1Denom,
		data,
	)
}
