package host

import (
	"cosmossdk.io/math"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	hostprovider "github.com/initia-labs/opinit-bots/provider/host"
	"github.com/initia-labs/opinit-bots/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	"github.com/pkg/errors"
)

func (h *Host) initiateDepositHandler(_ types.Context, args nodetypes.EventHandlerArgs) error {
	bridgeId, l1Sequence, from, to, l1Denom, l2Denom, amount, _, err := hostprovider.ParseMsgInitiateDeposit(args.EventAttributes)
	if err != nil {
		return errors.Wrap(err, "failed to parse initiate deposit event")
	}
	if bridgeId != h.BridgeId() {
		// pass other bridge deposit event
		return nil
	}
	if l1Sequence < h.initialL1Sequence {
		// pass old deposit event
		return nil
	}

	coinAmount, ok := math.NewIntFromString(amount)
	if !ok {
		return errors.New("invalid amount")
	}
	coin := sdk.NewCoin(l2Denom, coinAmount)

	deposit := challengertypes.NewDeposit(l1Sequence, args.BlockHeight, from, to, l1Denom, coin.String(), args.BlockTime)
	h.eventQueue = append(h.eventQueue, deposit)
	return nil
}
