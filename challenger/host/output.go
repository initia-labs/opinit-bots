package host

import (
	"encoding/base64"

	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	hostprovider "github.com/initia-labs/opinit-bots/provider/host"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
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

	output := challengertypes.NewOutput(l2BlockNumber, outputIndex, outputRoot[:], args.BlockTime)

	h.lastOutputIndex = outputIndex
	h.lastOutputTime = args.BlockTime
	h.eventQueue = append(h.eventQueue, output)
	h.outputPendingEventQueue = append(h.outputPendingEventQueue, output)

	ctx.Logger().Info("propose output",
		zap.Uint64("bridge_id", bridgeId),
		zap.String("proposer", proposer),
		zap.Uint64("output_index", outputIndex),
		zap.Int64("l2_block_number", l2BlockNumber),
		zap.String("output_root", base64.StdEncoding.EncodeToString(outputRoot)),
	)
	return nil
}
