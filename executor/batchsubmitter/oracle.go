package batchsubmitter

import (
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	childprovider "github.com/initia-labs/opinit-bots/provider/child"
	"github.com/initia-labs/opinit-bots/txutils"
	"github.com/pkg/errors"
)

// emptyOracleData converts the MsgUpdateOracle messages's data field to empty
// to decrease the size of the batch.
func (bs *BatchSubmitter) emptyOracleData(pbb *cmtproto.Block) (*cmtproto.Block, error) {
	txs := pbb.Data.GetTxs()
	if len(txs) == 0 {
		return pbb, nil
	}
	txBytes := txs[0]

	txConfig := bs.node.GetTxConfig()
	tx, err := txutils.DecodeTx(txConfig, txBytes)
	if err != nil {
		// ignore not registered tx in codec
		return pbb, nil
	}

	msgs := tx.GetMsgs()
	// oracle tx has only one message
	if len(msgs) != 1 {
		return pbb, nil
	}

	switch msg := msgs[0].(type) {
	case *opchildtypes.MsgUpdateOracle:
		msg.Data = []byte{}
	case *authz.MsgExec:
		if len(msg.Msgs) != 1 || msg.Msgs[0].TypeUrl != "/opinit.opchild.v1.MsgUpdateOracle" {
			return pbb, nil
		}
		oracleMsg := &opchildtypes.MsgUpdateOracle{}
		err = bs.node.Codec().UnpackAny(msg.Msgs[0], &oracleMsg)
		if err != nil {
			return nil, errors.Wrap(err, "failed to unpack oracle msg from authz msg")
		}
		oracleMsg.Data = []byte{}
		msgs[0], err = childprovider.CreateAuthzMsg(msg.Grantee, oracleMsg)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create authz msg")
		}
	}

	tx, err = txutils.ChangeMsgsFromTx(txConfig, tx, []sdk.Msg{msgs[0]})
	if err != nil {
		return nil, errors.Wrap(err, "failed to change msgs from tx")
	}
	convertedTxBytes, err := txutils.EncodeTx(txConfig, tx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode tx")
	}
	pbb.Data.Txs[0] = convertedTxBytes

	return pbb, nil
}
