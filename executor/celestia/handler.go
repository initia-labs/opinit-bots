package celestia

import (
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"
	"go.uber.org/zap"
)

func (c *Celestia) payForBlobsHandler(ctx types.Context, args nodetypes.EventHandlerArgs) error {
	var signer string
	var blobSizes string
	var namespaces string

	for _, attr := range args.EventAttributes {
		switch attr.Key {
		case "signer":
			signer = attr.Value
		case "blob_sizes": //
			blobSizes = attr.Value
		case "namespaces": // namespaces
			namespaces = attr.Value
		}
	}
	c.lastUpdatedBatchTime = args.BlockTime
	ctx.Logger().Info("record batch",
		zap.String("signer", signer),
		zap.String("blob_sizes", blobSizes),
		zap.String("namespaces", namespaces),
	)
	return c.SaveInternalStatus(c.db)
}
