package node

import (
	"encoding/json"

	"github.com/initia-labs/opinit-bots-go/db"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
	"github.com/initia-labs/opinit-bots-go/types"
)

func (n *Node) SaveSyncInfo() error {
	return n.db.Set(nodetypes.LastProcessedBlockHeight, db.FromInt64(n.lastProcessedBlockHeight))
}

func (n *Node) RawKVSyncInfo(height int64) types.KV {
	return types.KV{
		Key:   n.db.PrefixedKey(nodetypes.LastProcessedBlockHeight),
		Value: db.FromInt64(height),
	}
}

func (n *Node) loadSyncInfo() error {
	data, err := n.db.Get(nodetypes.LastProcessedBlockHeight)
	if err == db.ErrNotFound {
		return nil
	} else if err != nil {
		return err
	}
	n.lastProcessedBlockHeight = db.ToInt64(data)
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
	lastBytes := append(nodetypes.PrefixPendingTxs, 0xFF)
	iterErr := n.db.Iterate(nodetypes.PrefixPendingTxs, lastBytes, func(key, value []byte) (stop bool) {
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

func (n *Node) RawKVProcessedMsgs(processedData nodetypes.ProcessedData) (types.KV, error) {
	var data []byte
	var err error
	if len(processedData) > 0 {
		data, err = json.Marshal(processedData)
		if err != nil {
			return types.KV{}, err
		}
	}

	return types.KV{
		Key:   n.db.PrefixedKey(nodetypes.ProcessedDataKey),
		Value: data,
	}, nil
}

func (n *Node) SaveProcessedMsgs(processedData nodetypes.ProcessedData) error {
	data, err := json.Marshal(processedData)
	if err != nil {
		return err
	}
	return n.db.Set(nodetypes.ProcessedDataKey, data)
}

func (n *Node) loadProcessedMsgs() (processedData nodetypes.ProcessedData, err error) {
	data, err := n.db.Get(nodetypes.ProcessedDataKey)
	if err == db.ErrNotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &processedData)
	return processedData, nil
}
