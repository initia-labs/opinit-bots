package host

import (
	comettypes "github.com/cometbft/cometbft/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// If the relay oracle is enabled and the extended commit info contains votes, create a new MsgUpdateOracle message.
// Else return nil.
func (h *Host) oracleTxHandler(blockHeight uint64, extCommitBz comettypes.Tx) (sdk.Msg, error) {
	if !h.relayOracle {
		return nil, nil
	}
	return h.child.GetMsgUpdateOracle(
		blockHeight,
		extCommitBz,
	)
}
