package broadcaster

import (
	dbtypes "github.com/initia-labs/opinit-bots/db/types"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	"github.com/initia-labs/opinit-bots/types"

	"github.com/cosmos/cosmos-sdk/codec"
)

///////////////
// PendingTx //
///////////////

// SavePendingTx saves pending tx
func SavePendingTx(db types.BasicDB, pendingTx btypes.PendingTxInfo) error {
	data, err := pendingTx.Value()
	if err != nil {
		return err
	}
	return db.Set(pendingTx.Key(), data)
}

// DeletePendingTx deletes pending tx
func DeletePendingTx(db types.BasicDB, pendingTx btypes.PendingTxInfo) error {
	return db.Delete(pendingTx.Key())
}

// LoadPendingTxs loads all pending txs
func LoadPendingTxs(db types.DB) (txs []btypes.PendingTxInfo, err error) {
	iterErr := db.Iterate(dbtypes.AppendSplitter(btypes.PendingTxsPrefix), nil, func(_, value []byte) (stop bool, err error) {
		txInfo := btypes.PendingTxInfo{}
		err = txInfo.Unmarshal(value)
		if err != nil {
			return true, err
		}
		txs = append(txs, txInfo)
		return false, nil
	})
	if iterErr != nil {
		return nil, iterErr
	}
	return txs, err
}

// SavePendingTxs saves all pending txs
func SavePendingTxs(db types.BasicDB, txInfos []btypes.PendingTxInfo) error {
	for _, txInfo := range txInfos {
		if !txInfo.Save {
			continue
		}
		err := SavePendingTx(db, txInfo)
		if err != nil {
			return err
		}
	}
	return nil
}

// DeletePendingTxs deletes all pending txs
func DeletePendingTxs(db types.BasicDB, txInfos []btypes.PendingTxInfo) error {
	for _, txInfo := range txInfos {
		if err := DeletePendingTx(db, txInfo); err != nil {
			return err
		}
	}
	return nil
}

///////////////////
// ProcessedMsgs //
///////////////////

// SaveProcessedMsgs saves processed messages
func SaveProcessedMsgs(db types.BasicDB, cdc codec.Codec, processedMsgs btypes.ProcessedMsgs) error {
	data, err := processedMsgs.Value(cdc)
	if err != nil {
		return err
	}

	err = db.Set(processedMsgs.Key(), data)
	if err != nil {
		return err
	}
	return nil
}

// DeleteProcessedMsgs deletes processed messages
func DeleteProcessedMsgs(db types.BasicDB, processedMsgs btypes.ProcessedMsgs) error {
	return db.Delete(processedMsgs.Key())
}

// SaveProcessedMsgsBatch saves all processed messages in the batch
func SaveProcessedMsgsBatch(db types.BasicDB, cdc codec.Codec, processedMsgsBatch []btypes.ProcessedMsgs) error {
	for _, processedMsgs := range processedMsgsBatch {
		if !processedMsgs.Save {
			continue
		}

		data, err := processedMsgs.Value(cdc)
		if err != nil {
			return err
		}

		err = db.Set(processedMsgs.Key(), data)
		if err != nil {
			return err
		}
	}
	return nil
}

// LoadProcessedMsgsBatch loads all processed messages in the batch
func LoadProcessedMsgsBatch(db types.DB, cdc codec.Codec) (processedMsgsBatch []btypes.ProcessedMsgs, err error) {
	iterErr := db.Iterate(dbtypes.AppendSplitter(btypes.ProcessedMsgsPrefix), nil, func(_, value []byte) (stop bool, err error) {
		var processedMsgs btypes.ProcessedMsgs
		err = processedMsgs.UnmarshalInterfaceJSON(cdc, value)
		if err != nil {
			return true, err
		}
		processedMsgsBatch = append(processedMsgsBatch, processedMsgs)
		return false, nil
	})

	if iterErr != nil {
		return nil, iterErr
	}
	return processedMsgsBatch, nil
}

// DeleteProcessedMsgsBatch deletes all processed messages in the batch
func DeleteProcessedMsgsBatch(db types.BasicDB, processedMsgsBatch []btypes.ProcessedMsgs) error {
	for _, processedMsgs := range processedMsgsBatch {
		err := DeleteProcessedMsgs(db, processedMsgs)
		if err != nil {
			return err
		}
	}
	return nil
}
