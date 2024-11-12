package broadcaster

import (
	"go.uber.org/zap"

	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	"github.com/initia-labs/opinit-bots/types"
)

///////////////
// PendingTx //
///////////////

func (b Broadcaster) savePendingTx(sequence uint64, txInfo btypes.PendingTxInfo) error {
	data, err := txInfo.Marshal()
	if err != nil {
		return err
	}
	return b.db.Set(btypes.PrefixedPendingTx(sequence), data)
}

func (b Broadcaster) deletePendingTx(sequence uint64) error {
	return b.db.Delete(btypes.PrefixedPendingTx(sequence))
}

func (b Broadcaster) loadPendingTxs() (txs []btypes.PendingTxInfo, err error) {
	iterErr := b.db.PrefixedIterate(btypes.PendingTxsKey, nil, func(_, value []byte) (stop bool, err error) {
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

	b.logger.Debug("load pending txs", zap.Int("count", len(txs)))
	return txs, err
}

// PendingTxsToRawKV converts pending txs to raw kv pairs.
// If delete is true, it will return kv pairs for deletion (empty value).
func (b Broadcaster) PendingTxsToRawKV(txInfos []btypes.PendingTxInfo, delete bool) ([]types.RawKV, error) {
	kvs := make([]types.RawKV, 0, len(txInfos))
	for _, txInfo := range txInfos {
		var data []byte
		var err error

		if !delete && txInfo.Save {
			data, err = txInfo.Marshal()
			if err != nil {
				return nil, err
			}
		}
		kvs = append(kvs, types.RawKV{
			Key:   b.db.PrefixedKey(btypes.PrefixedPendingTx(txInfo.Sequence)),
			Value: data,
		})
	}
	return kvs, nil
}

///////////////////
// ProcessedMsgs //
///////////////////

// ProcessedMsgsToRawKV converts processed data to raw kv pairs.
// If delete is true, it will return kv pairs for deletion (empty value).
func (b Broadcaster) ProcessedMsgsToRawKV(ProcessedMsgs []btypes.ProcessedMsgs, delete bool) ([]types.RawKV, error) {
	kvs := make([]types.RawKV, 0, len(ProcessedMsgs))
	for _, processedMsgs := range ProcessedMsgs {
		var data []byte
		var err error

		if !delete && processedMsgs.Save {
			data, err = processedMsgs.MarshalInterfaceJSON(b.cdc)
			if err != nil {
				return nil, err
			}
		}
		kvs = append(kvs, types.RawKV{
			Key:   b.db.PrefixedKey(btypes.PrefixedProcessedMsgs(types.MustInt64ToUint64(processedMsgs.Timestamp))),
			Value: data,
		})
	}
	return kvs, nil
}

// currently no use case, but keep it for future use
// func (n *Broadcaster) saveProcessedMsgs(processedMsgs btypes.ProcessedMsgs) error {
// 	data, err := processedMsgs.Marshal()
// 	if err != nil {
// 		return err
// 	}
// 	return b.db.Set(btypes.PrefixedProcessedMsgs(uint64(processedMsgs.Timestamp)), data)
// }

func (b Broadcaster) loadProcessedMsgs() (ProcessedMsgs []btypes.ProcessedMsgs, err error) {
	iterErr := b.db.PrefixedIterate(btypes.ProcessedMsgsKey, nil, func(_, value []byte) (stop bool, err error) {
		var processedMsgs btypes.ProcessedMsgs
		err = processedMsgs.UnmarshalInterfaceJSON(b.cdc, value)
		if err != nil {
			return true, err
		}
		ProcessedMsgs = append(ProcessedMsgs, processedMsgs)
		return false, nil
	})

	if iterErr != nil {
		return nil, iterErr
	}
	b.logger.Debug("load pending processed msgs", zap.Int("count", len(ProcessedMsgs)))
	return ProcessedMsgs, nil
}

func (b Broadcaster) deleteProcessedMsgs(timestamp int64) error {
	return b.db.Delete(btypes.PrefixedProcessedMsgs(types.MustInt64ToUint64(timestamp)))
}
