package host

import (
	"errors"

	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	hostprovider "github.com/initia-labs/opinit-bots/provider/host"
	"github.com/initia-labs/opinit-bots/types"
	"go.uber.org/zap"
)

func (h *Host) recordBatchHandler(ctx types.Context, args nodetypes.EventHandlerArgs) error {
	hostAddress, err := h.GetAddressStr()
	if err != nil {
		if errors.Is(err, types.ErrKeyNotSet) {
			return nil
		}
		return nil
	}

	submitter, err := hostprovider.ParseMsgRecordBatch(args.EventAttributes)
	if err != nil {
		return err
	}

	if submitter != hostAddress {
		return nil
	}
	ctx.Logger().Info("record batch",
		zap.String("submitter", submitter),
	)
	return nil
}

func (h *Host) updateBatchInfoHandler(ctx types.Context, args nodetypes.EventHandlerArgs) error {
	bridgeId, submitter, chain, outputIndex, l2BlockNumber, err := hostprovider.ParseMsgUpdateBatchInfo(args.EventAttributes)
	if err != nil {
		return err
	}
	if bridgeId != h.BridgeId() {
		// pass other bridge deposit event
		return nil
	}

	ctx.Logger().Info("update batch info",
		zap.String("chain", chain),
		zap.String("submitter", submitter),
		zap.Uint64("output_index", outputIndex),
		zap.Int64("l2_block_number", l2BlockNumber),
	)

	h.batch.UpdateBatchInfo(chain, submitter, outputIndex, l2BlockNumber)
	return nil
}
