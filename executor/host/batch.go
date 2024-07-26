package host

import (
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
