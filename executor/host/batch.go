package host

import (
	"context"

	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	hostprovider "github.com/initia-labs/opinit-bots/provider/host"
	"go.uber.org/zap"
)

func (h *Host) recordBatchHandler(_ context.Context, args nodetypes.EventHandlerArgs) error {
	submitter, err := hostprovider.ParseMsgRecordBatch(args.EventAttributes)
	if err != nil {
		return err
	}
	hostAddress, err := h.GetAddressStr()
	if err != nil {
		return nil
	}
	if submitter != hostAddress {
		return nil
	}
	h.Logger().Info("record batch",
		zap.String("submitter", submitter),
	)
	return nil
}

func (h *Host) updateBatchInfoHandler(_ context.Context, args nodetypes.EventHandlerArgs) error {
	bridgeId, submitter, chain, outputIndex, l2BlockNumber, err := hostprovider.ParseMsgUpdateBatchInfo(args.EventAttributes)
	if err != nil {
		return err
	}
	if bridgeId != h.BridgeId() {
		// pass other bridge deposit event
		return nil
	}

	h.Logger().Info("update batch info",
		zap.String("chain", chain),
		zap.String("submitter", submitter),
		zap.Uint64("output_index", outputIndex),
		zap.Int64("l2_block_number", l2BlockNumber),
	)

	h.batch.UpdateBatchInfo(chain, submitter, outputIndex, l2BlockNumber)
	return nil
}
