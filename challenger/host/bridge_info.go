package host

import (
	"github.com/pkg/errors"
	"go.uber.org/zap"

	hostprovider "github.com/initia-labs/opinit-bots/provider/host"

	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"
)

func (h *Host) updateProposerHandler(ctx types.Context, args nodetypes.EventHandlerArgs) error {
	bridgeId, proposer, finalizedOutputIndex, finalizedL2BlockNumber, err := hostprovider.ParseMsgUpdateProposer(args.EventAttributes)
	if err != nil {
		return errors.Wrap(err, "failed to parse update proposer event")
	}
	if bridgeId != h.BridgeId() {
		return nil
	}

	ctx.Logger().Warn("update proposer",
		zap.Uint64("bridge_id", bridgeId),
		zap.String("proposer", proposer),
		zap.Uint64("finalized_output_index", finalizedOutputIndex),
		zap.Uint64("finalized_l2_block_number", finalizedL2BlockNumber),
	)

	h.UpdateProposer(proposer)
	return nil
}

func (h *Host) updateChallengerHandler(ctx types.Context, args nodetypes.EventHandlerArgs) error {
	bridgeId, challenger, finalizedOutputIndex, finalizedL2BlockNumber, err := hostprovider.ParseMsgUpdateChallenger(args.EventAttributes)
	if err != nil {
		return errors.Wrap(err, "failed to parse update challenger event")
	}
	if bridgeId != h.BridgeId() {
		return nil
	}

	ctx.Logger().Info("update challenger",
		zap.Uint64("bridge_id", bridgeId),
		zap.String("challenger", challenger),
		zap.Uint64("finalized_output_index", finalizedOutputIndex),
		zap.Uint64("finalized_l2_block_number", finalizedL2BlockNumber),
	)

	h.UpdateChallenger(challenger)
	return nil
}
