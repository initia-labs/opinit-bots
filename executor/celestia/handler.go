package celestia

import (
	"encoding/base64"

	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func (c *Celestia) payForBlobsHandler(ctx types.Context, args nodetypes.EventHandlerArgs) error {
	var signer string
	var blobSizes string
	var namespaces string

	for _, attr := range args.EventAttributes {
		switch attr.Key {
		case "c2lnbmVy": // signer
			value, err := base64.StdEncoding.DecodeString(attr.Value)
			if err != nil {
				return errors.Wrap(err, "failed to decode signer")
			}
			signer = string(value)
		case "YmxvYl9zaXplcw==": // blob_sizes
			value, err := base64.StdEncoding.DecodeString(attr.Value)
			if err != nil {
				return errors.Wrap(err, "failed to decode blob sizes")
			}
			blobSizes = string(value)
		case "bmFtZXNwYWNlcw==": // namespaces
			value, err := base64.StdEncoding.DecodeString(attr.Value)
			if err != nil {
				return errors.Wrap(err, "failed to decode namespaces")
			}
			namespaces = string(value)
		}
	}
	ctx.Logger().Info("record batch",
		zap.String("signer", signer),
		zap.String("blob_sizes", blobSizes),
		zap.String("namespaces", namespaces),
	)
	return nil
}
