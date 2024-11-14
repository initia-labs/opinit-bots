package batch

import (
	"encoding/json"

	dbtypes "github.com/initia-labs/opinit-bots/db/types"
	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	"github.com/initia-labs/opinit-bots/types"
)

var LocalBatchInfoKey = []byte("local_batch_info")

func GetLocalBatchInfo(db types.BasicDB) (executortypes.LocalBatchInfo, error) {
	val, err := db.Get(LocalBatchInfoKey)
	if err != nil {
		if err == dbtypes.ErrNotFound {
			return executortypes.LocalBatchInfo{}, nil
		}
		return executortypes.LocalBatchInfo{}, err
	}

	var localBatchInfo executortypes.LocalBatchInfo
	err = json.Unmarshal(val, &localBatchInfo)
	return localBatchInfo, err
}

func SaveLocalBatchInfo(db types.BasicDB, localBatchInfo executortypes.LocalBatchInfo) error {
	value, err := json.Marshal(&localBatchInfo)
	if err != nil {
		return err
	}
	return db.Set(LocalBatchInfoKey, value)
}
