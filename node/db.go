package node

import (
	dbtypes "github.com/initia-labs/opinit-bots/db/types"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"
	"go.uber.org/zap"
)

func (n *Node) SetSyncInfo(height int64) {
	n.lastProcessedBlockHeight = height
	if n.broadcaster != nil {
		n.broadcaster.SetSyncInfo(n.lastProcessedBlockHeight)
	}
}

func (n *Node) loadSyncInfo(processedHeight int64) error {
	data, err := n.db.Get(nodetypes.LastProcessedBlockHeightKey)
	if err == dbtypes.ErrNotFound {
		n.SetSyncInfo(processedHeight)
		n.startHeightInitialized = true
		n.logger.Info("initialize sync info", zap.Int64("start_height", processedHeight+1))
		return nil
	} else if err != nil {
		return err
	}

	lastSyncedHeight, err := dbtypes.ToInt64(data)
	if err != nil {
		return err
	}

	n.SetSyncInfo(lastSyncedHeight)
	n.logger.Debug("load sync info", zap.Int64("last_processed_height", n.lastProcessedBlockHeight))

	return nil
}

func (n Node) SaveSyncInfo(height int64) error {
	return n.db.Set(nodetypes.LastProcessedBlockHeightKey, dbtypes.FromUint64(types.MustInt64ToUint64(height)))
}

func (n Node) SyncInfoToRawKV(height int64) types.RawKV {
	return types.RawKV{
		Key:   n.db.PrefixedKey(nodetypes.LastProcessedBlockHeightKey),
		Value: dbtypes.FromUint64(types.MustInt64ToUint64(height)),
	}
}

func (n Node) DeleteSyncInfo() error {
	return n.db.Delete(nodetypes.LastProcessedBlockHeightKey)
}

func DeleteSyncInfo(db types.DB) error {
	return db.Delete(nodetypes.LastProcessedBlockHeightKey)
}

func DeleteProcessedMsgs(db types.DB) error {
	return db.PrefixedIterate(btypes.ProcessedMsgsKey, nil, func(key, _ []byte) (stop bool, err error) {
		err = db.Delete(key)
		if err != nil {
			return stop, err
		}
		return false, nil
	})
}

func DeletePendingTxs(db types.DB) error {
	return db.PrefixedIterate(btypes.PendingTxsKey, nil, func(key, _ []byte) (stop bool, err error) {
		err = db.Delete(key)
		if err != nil {
			return stop, err
		}
		return false, nil
	})
}
