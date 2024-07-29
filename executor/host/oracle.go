package host

import (
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"

	comettypes "github.com/cometbft/cometbft/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// If the relay oracle is enabled and the extended commit info contains votes, create a new MsgUpdateOracle message.
// Else return nil.
func (h *Host) oracleTxHandler(blockHeight uint64, extCommitBz comettypes.Tx) (sdk.Msg, error) {
	if !h.cfg.RelayOracle {
		return nil, nil
	}

	sender, err := h.child.GetAddressStr()
	if err != nil {
		return nil, err
	}

	msg := opchildtypes.NewMsgUpdateOracle(
		sender,
		blockHeight,
		extCommitBz,
	)

	err = msg.Validate(h.child.AccountCodec())
	if err != nil {
		return nil, err
	}
	return msg, nil
}
