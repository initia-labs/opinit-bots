package host

import (
	comettypes "github.com/cometbft/cometbft/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	hostprovider "github.com/initia-labs/opinit-bots/provider/host"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// If the relay oracle is enabled and the extended commit info contains votes, create a new MsgUpdateOracle message.
// Else return nil.
// Note: This is the legacy oracle method. When the new oracle relay is enabled, this method is disabled.
func (h *Host) oracleTxHandler(blockHeight int64, extCommitBz comettypes.Tx) (sdk.Msg, string, error) {
	if !h.OracleEnabled() {
		return nil, "", nil
	}

	// skip legacy oracle processing when the new oracle relay is enabled
	if h.oracleRelayEnabled {
		return nil, "", nil
	}

	return h.child.GetMsgUpdateOracle(
		blockHeight,
		extCommitBz,
	)
}

func (h *Host) updateOracleConfigHandler(ctx types.Context, args nodetypes.EventHandlerArgs) error {
	bridgeId, oracleEnabled, err := hostprovider.ParseMsgUpdateOracleConfig(args.EventAttributes)
	if err != nil {
		return errors.Wrap(err, "failed to parse update oracle config event")
	}
	if bridgeId != h.BridgeId() {
		return nil
	}

	ctx.Logger().Info("update oracle config",
		zap.Uint64("bridge_id", bridgeId),
		zap.Bool("oracle_enabled", oracleEnabled),
	)

	h.UpdateOracleEnabled(oracleEnabled)
	msg, sender, err := h.child.GetMsgSetBridgeInfo(bridgeId, h.BridgeInfo().BridgeConfig)
	if err != nil {
		return errors.Wrap(err, "failed to handle update challenger")
	} else if msg != nil {
		h.AppendMsgQueue(msg, sender)
	}
	return nil
}
