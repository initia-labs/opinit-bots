package celestia

import (
	"context"

	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"go.uber.org/zap"
)

func (c *Celestia) payForBlobsHandler(_ context.Context, args nodetypes.EventHandlerArgs) error {
	var signer string
	var blobSizes string
	var namespaces string

	for _, attr := range args.EventAttributes {
		switch attr.Key {
		case "c2lnbmVy": // signer
			signer = attr.Value
		case "YmxvYl9zaXplcw==": // blob_sizes
			blobSizes = attr.Value
		case "bmFtZXNwYWNlcw==": // namespaces
			namespaces = attr.Value
		}
	}
	c.logger.Info("record batch",
		zap.String("signer", signer),
		zap.String("blob_sizes", blobSizes),
		zap.String("namespaces", namespaces),
	)
	return nil
}
