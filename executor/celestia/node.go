package celestia

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pkg/errors"

	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	"github.com/initia-labs/opinit-bots/txutils"
	celestiatypes "github.com/initia-labs/opinit-bots/types/celestia"
)

// buildTxWithMessages creates a transaction from the given messages.
func (c *Celestia) BuildTxWithMessages(
	ctx context.Context,
	msgs []sdk.Msg,
) (
	txBytes []byte,
	txHash string,
	err error,
) {
	pfbMsgs := make([]sdk.Msg, 0, len(msgs))
	blobMsgs := make([]*celestiatypes.Blob, 0)
	for _, msg := range msgs {
		withBlobMsg, ok := msg.(*celestiatypes.MsgPayForBlobsWithBlob)
		if !ok {
			// not support other message types for now
			// only MsgPayForBlobsWithBlob in one tx
			return nil, "", fmt.Errorf("unsupported message type: %s", sdk.MsgTypeURL(msg))
		}
		pfbMsgs = append(pfbMsgs, withBlobMsg.MsgPayForBlobs)
		blobMsgs = append(blobMsgs, withBlobMsg.Blob)
	}

	broadcasterAccount, err := c.node.MustGetBroadcaster().AccountByIndex(0)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to calculate gas")
	}
	tx, err := broadcasterAccount.SimulateAndSignTx(ctx, pfbMsgs...)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to build unsigned tx")
	}
	txConfig := c.node.GetTxConfig()
	txBytes, err = txutils.EncodeTx(txConfig, tx)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to encode tx")
	}

	blobTx := celestiatypes.BlobTx{
		Tx:     txBytes,
		Blobs:  blobMsgs,
		TypeId: "BLOB",
	}
	blobTxBytes, err := blobTx.Marshal()
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to marshal blob tx")
	}

	return blobTxBytes, btypes.TxHash(txBytes), nil
}

func (c *Celestia) PendingTxToProcessedMsgs(
	txBytes []byte,
) ([]sdk.Msg, error) {
	txConfig := c.node.GetTxConfig()

	blobTx := &celestiatypes.BlobTx{}
	if err := blobTx.Unmarshal(txBytes); err == nil {
		pfbTx, err := txutils.DecodeTx(txConfig, blobTx.Tx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode blob tx")
		}
		pfbMsg := pfbTx.GetMsgs()[0]

		return []sdk.Msg{
			&celestiatypes.MsgPayForBlobsWithBlob{
				MsgPayForBlobs: pfbMsg.(*celestiatypes.MsgPayForBlobs),
				Blob:           blobTx.Blobs[0],
			},
		}, nil
	}

	tx, err := txutils.DecodeTx(txConfig, txBytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode tx")
	}
	return tx.GetMsgs(), nil
}
