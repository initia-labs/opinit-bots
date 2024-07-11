package host

import (
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"

	comettypes "github.com/cometbft/cometbft/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (h *Host) oracleTxHandler(blockHeight uint64, tx comettypes.Tx) (sdk.Msg, error) {
	sender, err := h.ac.BytesToString(h.child.GetAddress())
	if err != nil {
		return nil, err
	}

	msg := opchildtypes.NewMsgUpdateOracle(
		sender,
		blockHeight,
		tx,
	)
	err = msg.Validate(h.ac)
	if err != nil {
		return nil, err
	}
	return msg, nil
}
