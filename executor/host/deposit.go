package host

import (
	"cosmossdk.io/math"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	hostprovider "github.com/initia-labs/opinit-bots/provider/host"
	"github.com/initia-labs/opinit-bots/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pkg/errors"
)

func (h *Host) initiateDepositHandler(_ types.Context, args nodetypes.EventHandlerArgs) error {
	bridgeId, l1Sequence, from, to, l1Denom, l2Denom, amount, data, err := hostprovider.ParseMsgInitiateDeposit(args.EventAttributes)
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
		return errors.New("invalid coin amount")
	}
	coin := sdk.NewCoin(l2Denom, coinAmount)

	msg, sender, err := h.child.GetMsgFinalizeTokenDeposit(
		from,
		to,
		coin,
		l1Sequence,
		args.BlockHeight,
		l1Denom,
		data,
	)
	if err != nil {
		return errors.Wrap(err, "failed to handle initiate deposit")
	} else if msg != nil {
		h.AppendMsgQueue(msg, sender)
	}
	return nil
}
