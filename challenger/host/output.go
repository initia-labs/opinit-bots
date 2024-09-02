package host

import (
	"context"
	"time"

	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	hostprovider "github.com/initia-labs/opinit-bots/provider/host"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
)

func (h *Host) proposeOutputHandler(_ context.Context, args nodetypes.EventHandlerArgs) error {
	bridgeId, l2BlockNumber, outputIndex, _, outputRoot, err := hostprovider.ParseMsgProposeOutput(args.EventAttributes)
	if err != nil {
		return err
	}
	if bridgeId != uint64(h.BridgeId()) {
		// pass other bridge output proposal event
		return nil
	}
	return h.handleProposeOutput(outputIndex, l2BlockNumber, outputRoot, args.BlockTime)
}

func (h *Host) handleProposeOutput(outputIndex uint64, l2BlockNumber uint64, outputRoot []byte, blockTime time.Time) error {
	output := challengertypes.NewOutput(l2BlockNumber, outputIndex, outputRoot[:], blockTime)
	h.eventQueue = append(h.eventQueue, output)
	return nil
}
