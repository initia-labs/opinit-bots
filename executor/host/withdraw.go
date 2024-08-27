package host

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"strconv"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"go.uber.org/zap"
)

func (h *Host) proposeOutputHandler(_ context.Context, args nodetypes.EventHandlerArgs) error {
	var bridgeId, l2BlockNumber, outputIndex uint64
	var proposer string
	var outputRoot []byte
	var err error

	for _, attr := range args.EventAttributes {
		switch attr.Key {
		case ophosttypes.AttributeKeyProposer:
			proposer = attr.Value
		case ophosttypes.AttributeKeyBridgeId:
			bridgeId, err = strconv.ParseUint(attr.Value, 10, 64)
			if err != nil {
				return err
			}
			if bridgeId != uint64(h.BridgeId()) {
				// pass other bridge output proposal event
				return nil
			}
		case ophosttypes.AttributeKeyOutputIndex:
			outputIndex, err = strconv.ParseUint(attr.Value, 10, 64)
			if err != nil {
				return err
			}
		case ophosttypes.AttributeKeyL2BlockNumber:
			l2BlockNumber, err = strconv.ParseUint(attr.Value, 10, 64)
			if err != nil {
				return err
			}
		case ophosttypes.AttributeKeyOutputRoot:
			outputRoot, err = hex.DecodeString(attr.Value)
			if err != nil {
				return err
			}
		}
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
	var bridgeId uint64
	var outputIndex, l2Sequence uint64
	var from, to, l1Denom, l2Denom, amount string
	var err error

	for _, attr := range args.EventAttributes {
		switch attr.Key {
		case ophosttypes.AttributeKeyBridgeId:
			bridgeId, err = strconv.ParseUint(attr.Value, 10, 64)
			if err != nil {
				return err
			}
			if bridgeId != uint64(h.BridgeId()) {
				// pass other bridge withdrawal event
				return nil
			}
		case ophosttypes.AttributeKeyOutputIndex:
			outputIndex, err = strconv.ParseUint(attr.Value, 10, 64)
			if err != nil {
				return err
			}
		case ophosttypes.AttributeKeyL2Sequence:
			l2Sequence, err = strconv.ParseUint(attr.Value, 10, 64)
			if err != nil {
				return err
			}
		case ophosttypes.AttributeKeyFrom:
			from = attr.Value
		case ophosttypes.AttributeKeyTo:
			to = attr.Value
		case ophosttypes.AttributeKeyL1Denom:
			l1Denom = attr.Value
		case ophosttypes.AttributeKeyL2Denom:
			l2Denom = attr.Value
		case ophosttypes.AttributeKeyAmount:
			amount = attr.Value
		}
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
