package celestia

import (
	"context"
	"encoding/base64"

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
			value, err := base64.StdEncoding.DecodeString(attr.Value)
			if err != nil {
				return err
			}
			signer = string(value)
		case "YmxvYl9zaXplcw==": // blob_sizes
			value, err := base64.StdEncoding.DecodeString(attr.Value)
			if err != nil {
				return err
			}
			blobSizes = string(value)
		case "bmFtZXNwYWNlcw==": // namespaces
			value, err := base64.StdEncoding.DecodeString(attr.Value)
			if err != nil {
				return err
			}
			namespaces = string(value)
		}
	}
	c.logger.Info("record batch",
		zap.String("signer", signer),
		zap.String("blob_sizes", blobSizes),
		zap.String("namespaces", namespaces),
	)
	return nil
}
