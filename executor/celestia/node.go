package celestia

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"

	btypes "github.com/initia-labs/opinit-bots-go/node/broadcaster/types"
	celestiatypes "github.com/initia-labs/opinit-bots-go/types/celestia"
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
			return nil, "", err
		}
		pfbMsgs = append(pfbMsgs, withBlobMsg.MsgPayForBlobs)
		blobMsgs = append(blobMsgs, withBlobMsg.Blob)
	}

	b := c.node.MustGetBroadcaster()
	txf := b.GetTxf()

	_, adjusted, err := b.CalculateGas(ctx, txf, pfbMsgs...)
	if err != nil {
		return nil, "", err
	}

	txf = txf.WithGas(adjusted)
	txb, err := txf.BuildUnsignedTx(pfbMsgs...)
	if err != nil {
		return nil, "", err
	}

	if err = tx.Sign(ctx, txf, b.KeyName(), txb, false); err != nil {
		return nil, "", err
	}

	tx := txb.GetTx()
	txBytes, err = b.EncodeTx(tx)
	if err != nil {
		return nil, "", err
	}

	blobTx := celestiatypes.BlobTx{
		Tx:     txBytes,
		Blobs:  blobMsgs,
		TypeId: "BLOB",
	}
	blobTxBytes, err := blobTx.Marshal()
	if err != nil {
		return nil, "", err
	}

	return blobTxBytes, btypes.TxHash(txBytes), nil
}

func (c *Celestia) PendingTxToProcessedMsgs(
	txBytes []byte,
) ([]sdk.Msg, error) {
	b := c.node.MustGetBroadcaster()

	blobTx := &celestiatypes.BlobTx{}
	if err := blobTx.Unmarshal(txBytes); err == nil {
		pfbTx, err := b.DecodeTx(blobTx.Tx)
		if err != nil {
			return nil, err
		}
		pfbMsg := pfbTx.GetMsgs()[0]

		return []sdk.Msg{
			&celestiatypes.MsgPayForBlobsWithBlob{
				MsgPayForBlobs: pfbMsg.(*celestiatypes.MsgPayForBlobs),
				Blob:           blobTx.Blobs[0],
			},
		}, nil
	}

	tx, err := b.DecodeTx(txBytes)
	if err != nil {
		return nil, err
	}
	return tx.GetMsgs(), nil
}
