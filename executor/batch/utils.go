package batch

import (
	"encoding/binary"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	"github.com/initia-labs/opinit-bots-go/txutils"
)

// prependLength prepends the length of the data to the data.
func prependLength(data []byte) []byte {
	lengthBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(lengthBytes, uint64(len(lengthBytes)))
	return append(lengthBytes, data...)
}

// emptyOracleData converts the MsgUpdateOracle messages's data field to empty
// to decrease the size of the batch.
func (bs *BatchSubmitter) emptyOracleData(pbb *cmtproto.Block) ([]byte, error) {
	for i, txBytes := range pbb.Data.GetTxs() {
		txConfig := bs.node.GetTxConfig()
		tx, err := txutils.DecodeTx(txConfig, txBytes)
		if err != nil {
			// ignore not registered tx in codec
			continue
		}

		msgs := tx.GetMsgs()
		if len(msgs) != 1 {
			continue
		}

		if msg, ok := msgs[0].(*opchildtypes.MsgUpdateOracle); ok {
			msg.Data = []byte{}
			tx, err := txutils.ChangeMsgsFromTx(txConfig, tx, []sdk.Msg{msg})
			if err != nil {
				return nil, err
			}
			convertedTxBytes, err := txutils.EncodeTx(txConfig, tx)
			if err != nil {
				return nil, err
			}
			pbb.Data.Txs[i] = convertedTxBytes
		}
	}

	// convert block to bytes
	blockBytes, err := proto.Marshal(pbb)
	if err != nil {
		return nil, err
	}
	return blockBytes, nil
}
