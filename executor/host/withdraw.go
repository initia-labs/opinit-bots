package host

import (
	"context"
	"encoding/base64"

	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	hostprovider "github.com/initia-labs/opinit-bots/provider/host"
	"go.uber.org/zap"
)

func (h *Host) proposeOutputHandler(_ context.Context, args nodetypes.EventHandlerArgs) error {
	bridgeId, l2BlockNumber, outputIndex, proposer, outputRoot, err := hostprovider.ParseMsgProposeOutput(args.EventAttributes)
	if err != nil {
		return err
	}
	if bridgeId != uint64(h.BridgeId()) {
		// pass other bridge output proposal event
		return nil
	}

	h.handleProposeOutput(bridgeId, proposer, outputIndex, l2BlockNumber, outputRoot)
	h.lastProposedOutputIndex = outputIndex
	h.lastProposedOutputL2BlockNumber = l2BlockNumber
	return nil
}

func (h *Host) handleProposeOutput(bridgeId uint64, proposer string, outputIndex uint64, l2BlockNumber uint64, outputRoot []byte) {
	h.Logger().Info("propose output",
		zap.Uint64("bridge_id", bridgeId),
		zap.String("proposer", proposer),
		zap.Uint64("output_index", outputIndex),
		zap.Uint64("l2_block_number", l2BlockNumber),
		zap.String("output_root", base64.StdEncoding.EncodeToString(outputRoot)),
	)
}

func (h *Host) finalizeWithdrawalHandler(_ context.Context, args nodetypes.EventHandlerArgs) error {
	bridgeId, outputIndex, l2Sequence, from, to, l1Denom, l2Denom, amount, err := hostprovider.ParseMsgFinalizeWithdrawal(args.EventAttributes)
	if err != nil {
		return err
	}
	if bridgeId != uint64(h.BridgeId()) {
		// pass other bridge withdrawal event
		return nil
	}

	h.handleFinalizeWithdrawal(bridgeId, outputIndex, l2Sequence, from, to, l1Denom, l2Denom, amount)
	return nil
}

func (h *Host) handleFinalizeWithdrawal(bridgeId uint64, outputIndex uint64, l2Sequence uint64, from string, to string, l1Denom string, l2Denom string, amount string) {
	h.Logger().Info("finalize withdrawal",
		zap.Uint64("bridge_id", bridgeId),
		zap.Uint64("output_index", outputIndex),
		zap.Uint64("l2_sequence", l2Sequence),
		zap.String("from", from),
		zap.String("to", to),
		zap.String("l1_denom", l1Denom),
		zap.String("l2_denom", l2Denom),
		zap.String("amount", amount),
	)
}
