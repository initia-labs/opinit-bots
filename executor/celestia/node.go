package celestia

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	node "github.com/initia-labs/opinit-bots-go/node"
	celestiatypes "github.com/initia-labs/opinit-bots-go/types/celestia"
)

// buildTxWithMessages creates a transaction from the given messages.
func CelestiaBuildTxWithMessages(
	n *node.Node,
	ctx context.Context,
	msgs []sdk.Msg,
) (
	txBytes []byte,
	err error,
) {
	pfbMsgs := make([]sdk.Msg, 0, len(msgs))
	blobMsgs := make([]*celestiatypes.Blob, 0)
	for _, msg := range msgs {
		withBlobMsg, ok := msg.(*celestiatypes.MsgPayForBlobsWithBlob)
		if !ok {
			return nil, err
		}
		pfbMsgs = append(pfbMsgs, withBlobMsg.MsgPayForBlobs)
		blobMsgs = append(blobMsgs, withBlobMsg.Blob)
	}

	txf := n.GetTxf()
	_, adjusted, err := n.CalculateGas(ctx, txf, pfbMsgs...)
	if err != nil {
		return nil, err
	}

	txf = txf.WithGas(adjusted)
	txb, err := txf.BuildUnsignedTx(msgs...)
	if err != nil {
		return nil, err
	}

	if err = tx.Sign(ctx, txf, n.KeyName(), txb, false); err != nil {
		return nil, err
	}

	tx := txb.GetTx()
	txBytes, err = n.EncodeTx(tx)
	if err != nil {
		return nil, err
	}

	blobTx := celestiatypes.BlobTx{
		Tx:     txBytes,
		Blobs:  blobMsgs,
		TypeId: "BLOB",
	}
	return blobTx.Marshal()
}
