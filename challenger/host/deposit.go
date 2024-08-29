package host

import (
	"context"
	"errors"
	"time"

	"cosmossdk.io/math"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	hostprovider "github.com/initia-labs/opinit-bots/provider/host"

	sdk "github.com/cosmos/cosmos-sdk/types"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
)

func (h *Host) initiateDepositHandler(_ context.Context, args nodetypes.EventHandlerArgs) error {
	bridgeId, l1Sequence, from, to, l1Denom, l2Denom, amount, _, err := hostprovider.ParseMsgInitiateDeposit(args.EventAttributes)
	if err != nil {
		return err
	}
	if bridgeId != uint64(h.BridgeId()) {
		// pass other bridge deposit event
		return nil
	}

	return h.handleInitiateDeposit(
		l1Sequence,
		args.BlockHeight,
		args.BlockTime,
		from,
		to,
		l1Denom,
		l2Denom,
		amount,
	)
}

func (h *Host) handleInitiateDeposit(
	l1Sequence uint64,
	blockHeight uint64,
	blockTime time.Time,
	from string,
	to string,
	l1Denom string,
	l2Denom string,
	amount string,
) error {
	coinAmount, ok := math.NewIntFromString(amount)
	if !ok {
		return errors.New("invalid amount")
	}
	coin := sdk.NewCoin(l2Denom, coinAmount)

	deposit := challengertypes.NewDeposit(l1Sequence, blockHeight, blockTime, from, to, l1Denom, coin.String())
	h.elemQueue = append(h.elemQueue, challengertypes.ChallengeElem{
		Node:  h.NodeType(),
		Id:    deposit.Sequence,
		Event: deposit,
	})
	return nil
}
