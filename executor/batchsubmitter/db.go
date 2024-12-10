package batchsubmitter

import (
	dbtypes "github.com/initia-labs/opinit-bots/db/types"
	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"
)

// GetLocalBatchInfo returns the local batch info from the given db.
// If the local batch info is not found, it returns an empty struct.
func GetLocalBatchInfo(db types.BasicDB) (executortypes.LocalBatchInfo, error) {
	val, err := db.Get(executortypes.LocalBatchInfoKey)
	if err != nil {
		if errors.Is(err, dbtypes.ErrNotFound) {
			return executortypes.LocalBatchInfo{}, nil
		}
		return executortypes.LocalBatchInfo{}, err
	}

	localBatchInfo := executortypes.LocalBatchInfo{}
	err = localBatchInfo.Unmarshal(val)
	return localBatchInfo, err
}

// SaveLocalBatchInfo saves the local batch info to the given db.
func SaveLocalBatchInfo(db types.BasicDB, localBatchInfo executortypes.LocalBatchInfo) error {
	bz, err := localBatchInfo.Value()
	if err != nil {
		return err
	}
	return db.Set(localBatchInfo.Key(), bz)
}
