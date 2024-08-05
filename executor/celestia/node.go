package celestia

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	node "github.com/initia-labs/opinit-bots-go/node"
	celestiatypes "github.com/initia-labs/opinit-bots-go/types/celestia"
)

// buildTxWithMessages creates a transaction from the given messages.
func BuildTxWithMessages(
	n *node.Node,
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

	txf := n.GetTxf()
	_, adjusted, err := n.CalculateGas(ctx, txf, pfbMsgs...)
	if err != nil {
		return nil, "", err
	}

	txf = txf.WithGas(adjusted)
	txb, err := txf.BuildUnsignedTx(pfbMsgs...)
	if err != nil {
		return nil, "", err
	}

	if err = tx.Sign(ctx, txf, n.KeyName(), txb, false); err != nil {
		return nil, "", err
	}

	tx := txb.GetTx()
	txBytes, err = n.EncodeTx(tx)
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

	return blobTxBytes, node.TxHash(txBytes), nil
}

func PendingTxToProcessedMsgs(
	n *node.Node,
	txBytes []byte,
) ([]sdk.Msg, error) {
	blobTx := &celestiatypes.BlobTx{}
	if err := blobTx.Unmarshal(txBytes); err == nil {
		pfbTx, err := n.DecodeTx(blobTx.Tx)
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

	tx, err := n.DecodeTx(txBytes)
	if err != nil {
		return nil, err
	}
	return tx.GetMsgs(), nil
}
