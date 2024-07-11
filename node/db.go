package node

import (
	dbtypes "github.com/initia-labs/opinit-bots-go/db/types"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
	"github.com/initia-labs/opinit-bots-go/types"
	"go.uber.org/zap"
)

func (n *Node) SaveSyncInfo() error {
	return n.db.Set(nodetypes.LastProcessedBlockHeightKey, dbtypes.FromUint64(n.lastProcessedBlockHeight))
}

func (n *Node) RawKVSyncInfo(height uint64) types.KV {
	return types.KV{
		Key:   n.db.PrefixedKey(nodetypes.LastProcessedBlockHeightKey),
		Value: dbtypes.FromUint64(height),
	}
}

func (n *Node) loadSyncInfo() error {
	data, err := n.db.Get(nodetypes.LastProcessedBlockHeightKey)
	if err == dbtypes.ErrNotFound {
		return nil
	} else if err != nil {
		return err
	}
	n.lastProcessedBlockHeight = dbtypes.ToUint64(data)
	n.logger.Info("load sync info", zap.Uint64("last_processed_height", n.lastProcessedBlockHeight))
	return nil
}

func (n Node) savePendingTx(sequence uint64, txInfo nodetypes.PendingTxInfo) error {
	data, err := txInfo.Marshal()
	if err != nil {
		return err
	}
	return n.db.Set(nodetypes.PrefixedPendingTx(sequence), data)
}

func (n Node) deletePendingTx(sequence uint64) error {
	return n.db.Delete(nodetypes.PrefixedPendingTx(sequence))
}

func (n *Node) loadPendingTxs() (txs []nodetypes.PendingTxInfo, err error) {
	iterErr := n.db.Iterate(nodetypes.PendingTxsKey, nodetypes.LastPendingTxKey, func(key, value []byte) (stop bool) {
		txInfo := nodetypes.PendingTxInfo{}
		err = txInfo.Unmarshal(value)
		if err != nil {
			return true
		}
		txs = append(txs, txInfo)
		return false
	})

	if iterErr != nil {
		return nil, iterErr
	}
	n.logger.Info("load pending txs", zap.Int("count", len(txs)))
	return txs, err
}

func (n *Node) RawKVPendingTxs(txInfos []nodetypes.PendingTxInfo, delete bool) ([]types.KV, error) {
	kvs := make([]types.KV, 0, len(txInfos))
	for _, txInfo := range txInfos {
		if !txInfo.Save {
			continue
		}

		var data []byte
		var err error

		if !delete {
			data, err = txInfo.Marshal()
			if err != nil {
				return nil, err
			}
		}
		kvs = append(kvs, types.KV{
			Key:   n.db.PrefixedKey(nodetypes.PrefixedPendingTx(txInfo.Sequence)),
			Value: data,
		})
	}
	return kvs, nil
}

func (n *Node) RawKVProcessedData(processedData []nodetypes.ProcessedMsgs, delete bool) ([]types.KV, error) {
	kvs := make([]types.KV, 0, len(processedData))
	for _, processedMsgs := range processedData {
		if !processedMsgs.Save {
			continue
		}

		var data []byte
		var err error

		if !delete {
			data, err = processedMsgs.Marshal()
			if err != nil {
				return nil, err
			}
		}
		kvs = append(kvs, types.KV{
			Key:   n.db.PrefixedKey(nodetypes.PrefixedProcessedMsgs(uint64(processedMsgs.Timestamp))),
			Value: data,
		})
	}
	return kvs, nil
}

func (n *Node) saveProcessedMsgs(processedMsgs nodetypes.ProcessedMsgs) error {
	data, err := processedMsgs.Marshal()
	if err != nil {
		return err
	}
	return n.db.Set(nodetypes.PrefixedProcessedMsgs(uint64(processedMsgs.Timestamp)), data)
}

func (n *Node) loadProcessedData() (processedData []nodetypes.ProcessedMsgs, err error) {
	iterErr := n.db.Iterate(nodetypes.ProcessedMsgsKey, nodetypes.LastProcessedMsgsKey, func(key, value []byte) (stop bool) {
		processedMsgs := nodetypes.ProcessedMsgs{}
		err = processedMsgs.Unmarshal(value)
		if err != nil {
			return true
		}
		processedData = append(processedData, processedMsgs)
		return false
	})

	if iterErr != nil {
		return nil, iterErr
	}
	n.logger.Info("load pending processed msgs", zap.Int("count", len(processedData)))
	return processedData, nil
}

func (n *Node) deleteProcessedMsgs(timestamp int64) error {
	return n.db.Delete(nodetypes.PrefixedProcessedMsgs(uint64(timestamp)))
}
