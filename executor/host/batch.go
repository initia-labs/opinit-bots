package host

import (
	"strconv"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
	"go.uber.org/zap"
)

func (h *Host) recordBatchHandler(args nodetypes.EventHandlerArgs) error {
	var submitter string
	for _, attr := range args.EventAttributes {
		switch attr.Key {
		case ophosttypes.AttributeKeySubmitter:
			submitter = attr.Value
			hostAddress, err := h.GetAddressStr()
			if err != nil {
				return nil
			}
			if submitter != hostAddress {
				return nil
			}
		}
	}
	h.logger.Info("record batch",
		zap.String("submitter", submitter),
	)
	return nil
}

func (h *Host) updateBatchInfoHandler(args nodetypes.EventHandlerArgs) error {
	var bridgeId uint64
	var submitter, chain string
	var outputIndex, l2BlockNumber uint64
	var err error
	for _, attr := range args.EventAttributes {
		switch attr.Key {
		case ophosttypes.AttributeKeyBridgeId:
			bridgeId, err = strconv.ParseUint(attr.Value, 10, 64)
			if err != nil {
				return err
			}
			if bridgeId != uint64(h.bridgeId) {
				// pass other bridge deposit event
				return nil
			}
		case ophosttypes.AttributeKeyBatchChainType:
			chain = attr.Value
		case ophosttypes.AttributeKeyBatchSubmitter:
			submitter = attr.Value
		case ophosttypes.AttributeKeyFinalizedOutputIndex:
			outputIndex, err = strconv.ParseUint(attr.Value, 10, 64)
			if err != nil {
				return err
			}
		case ophosttypes.AttributeKeyFinalizedL2BlockNumber:
			l2BlockNumber, err = strconv.ParseUint(attr.Value, 10, 64)
			if err != nil {
				return err
			}
		}
	}
	h.logger.Info("update batch info",
		zap.String("chain", chain),
		zap.String("submitter", submitter),
		zap.Uint64("output_index", outputIndex),
		zap.Uint64("l2_block_number", l2BlockNumber),
	)

	h.batch.UpdateBatchInfo(chain, submitter, outputIndex, l2BlockNumber)
	return nil
}
