package node

import (
	dbtypes "github.com/initia-labs/opinit-bots/db/types"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"
)

// GetSyncInfo gets the synced height
func GetSyncInfo(db types.BasicDB) (int64, error) {
	loadedHeightBytes, err := db.Get(nodetypes.SyncedHeightKey)
	if err != nil {
		return 0, errors.Wrap(err, "failed to load sync info")
	}

	syncedHeight, err := nodetypes.UnmarshalHeight(loadedHeightBytes)
	if err != nil {
		return 0, errors.Wrap(err, "failed to deserialize height")
	}
	return syncedHeight, nil
}

// SetSyncInfo sets the synced height
func SetSyncedHeight(db types.BasicDB, height int64) error {
	return db.Set(nodetypes.SyncedHeightKey, nodetypes.MarshalHeight(height))
}

// DeleteSyncInfo deletes the synced height
func DeleteSyncedHeight(db types.BasicDB) error {
	return db.Delete(nodetypes.SyncedHeightKey)
}

// DeleteProcessedMsgs deletes all processed messages
func DeleteProcessedMsgs(db types.DB) error {
	return db.Iterate(dbtypes.AppendSplitter(btypes.ProcessedMsgsPrefix), nil, func(key, _ []byte) (stop bool, err error) {
		err = db.Delete(key)
		if err != nil {
			return stop, err
		}
		return false, nil
	})
}

// DeletePendingTxs deletes all pending txs
func DeletePendingTxs(db types.DB) error {
	return db.Iterate(dbtypes.AppendSplitter(btypes.PendingTxsPrefix), nil, func(key, _ []byte) (stop bool, err error) {
		err = db.Delete(key)
		if err != nil {
			return stop, err
		}
		return false, nil
	})
}
