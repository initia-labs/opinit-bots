package host

import (
	"encoding/base64"

	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	hostprovider "github.com/initia-labs/opinit-bots/provider/host"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func (h *Host) proposeOutputHandler(ctx types.Context, args nodetypes.EventHandlerArgs) error {
	bridgeId, l2BlockNumber, outputIndex, proposer, outputRoot, err := hostprovider.ParseMsgProposeOutput(args.EventAttributes)
	if err != nil {
		return errors.Wrap(err, "failed to parse propose output event")
	}
	if bridgeId != h.BridgeId() {
		// pass other bridge output proposal event
		return nil
	}

	h.lastProposedOutputIndex = outputIndex
	h.lastProposedOutputL2BlockNumber = l2BlockNumber
	h.lastProposedOutputTime = args.BlockTime

	ctx.Logger().Info("propose output",
		zap.Uint64("bridge_id", bridgeId),
		zap.String("proposer", proposer),
		zap.Uint64("output_index", outputIndex),
		zap.Int64("l2_block_number", l2BlockNumber),
		zap.String("output_root", base64.StdEncoding.EncodeToString(outputRoot)),
	)
	return nil
}

func (h *Host) finalizeWithdrawalHandler(ctx types.Context, args nodetypes.EventHandlerArgs) error {
	bridgeId, outputIndex, l2Sequence, from, to, l1Denom, l2Denom, amount, err := hostprovider.ParseMsgFinalizeWithdrawal(args.EventAttributes)
	if err != nil {
		return errors.Wrap(err, "failed to parse finalize withdrawal event")
	}
	if bridgeId != h.BridgeId() {
		// pass other bridge withdrawal event
		return nil
	}

	ctx.Logger().Info("finalize withdrawal",
		zap.Uint64("bridge_id", bridgeId),
		zap.Uint64("output_index", outputIndex),
		zap.Uint64("l2_sequence", l2Sequence),
		zap.String("from", from),
		zap.String("to", to),
		zap.String("l1_denom", l1Denom),
		zap.String("l2_denom", l2Denom),
		zap.String("amount", amount),
	)
	return nil
}
