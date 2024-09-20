package node

import (
	dbtypes "github.com/initia-labs/opinit-bots/db/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"
	"go.uber.org/zap"
)

func (n *Node) SetSyncInfo(height uint64) {
	n.lastProcessedBlockHeight = height
	if n.broadcaster != nil {
		n.broadcaster.SetSyncInfo(n.lastProcessedBlockHeight)
	}
}

func (n *Node) loadSyncInfo(startHeight uint64) error {
	data, err := n.db.Get(nodetypes.LastProcessedBlockHeightKey)
	if err == dbtypes.ErrNotFound {
		n.SetSyncInfo(startHeight)
		n.startHeightInitialized = true
		n.logger.Info("initialize sync info", zap.Uint64("start_height", startHeight+1))
		return nil
	} else if err != nil {
		return err
	}

	lastSyncedHeight, err := dbtypes.ToUint64(data)
	if err != nil {
		return err
	}

	n.SetSyncInfo(lastSyncedHeight)
	n.logger.Debug("load sync info", zap.Uint64("last_processed_height", n.lastProcessedBlockHeight))

	return nil
}

func (n Node) SaveSyncInfo(height uint64) error {
	return n.db.Set(nodetypes.LastProcessedBlockHeightKey, dbtypes.FromUint64(height))
}

func (n Node) SyncInfoToRawKV(height uint64) types.RawKV {
	return types.RawKV{
		Key:   n.db.PrefixedKey(nodetypes.LastProcessedBlockHeightKey),
		Value: dbtypes.FromUint64(height),
	}
}

func (n Node) DeleteSyncInfo() error {
	return n.db.Delete(nodetypes.LastProcessedBlockHeightKey)
}

func DeleteSyncInfo(db types.DB) error {
	return db.Delete(nodetypes.LastProcessedBlockHeightKey)
}
