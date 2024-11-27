package batch

import (
	dbtypes "github.com/initia-labs/opinit-bots/db/types"
	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"
)

var LocalBatchInfoKey = []byte("local_batch_info")

func GetLocalBatchInfo(db types.BasicDB) (executortypes.LocalBatchInfo, error) {
	val, err := db.Get(LocalBatchInfoKey)
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

func SaveLocalBatchInfo(db types.BasicDB, localBatchInfo executortypes.LocalBatchInfo) error {
	value, err := localBatchInfo.Marshal()
	if err != nil {
		return err
	}
	return db.Set(LocalBatchInfoKey, value)
}
