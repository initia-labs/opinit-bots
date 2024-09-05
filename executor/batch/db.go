package batch

import (
	"encoding/json"

	dbtypes "github.com/initia-labs/opinit-bots/db/types"
	"github.com/initia-labs/opinit-bots/types"
)

var LocalBatchInfoKey = []byte("local_batch_info")

func (bs *BatchSubmitter) loadLocalBatchInfo() error {
	val, err := bs.db.Get(LocalBatchInfoKey)
	if err != nil {
		if err == dbtypes.ErrNotFound {
			return nil
		}
		return err
	}
	return json.Unmarshal(val, &bs.localBatchInfo)
}

func (bs *BatchSubmitter) localBatchInfoToRawKV() (types.RawKV, error) {
	value, err := json.Marshal(bs.localBatchInfo)
	if err != nil {
		return types.RawKV{}, err
	}
	return types.RawKV{
		Key:   bs.db.PrefixedKey(LocalBatchInfoKey),
		Value: value,
	}, nil
}

func (bs *BatchSubmitter) saveLocalBatchInfo() error {
	value, err := json.Marshal(bs.localBatchInfo)
	if err != nil {
		return err
	}
	return bs.db.Set(LocalBatchInfoKey, value)
}
