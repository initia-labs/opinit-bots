package executor

import (
	"fmt"

	dbtypes "github.com/initia-labs/opinit-bots/db/types"
	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	"github.com/initia-labs/opinit-bots/node"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"
)

func ResetHeights(db types.DB) error {
	dbNames := []string{
		types.HostName,
		types.ChildName,
		types.BatchName,
		types.DAName,
	}
	for _, dbName := range dbNames {
		if err := ResetHeight(db, dbName); err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to reset height for %s", dbName))
		}
	}
	return nil
}

func ResetHeight(db types.DB, nodeName string) error {
	if nodeName != types.HostName &&
		nodeName != types.ChildName &&
		nodeName != types.BatchName &&
		nodeName != types.DAName {
		return errors.New("unknown node name")
	}
	nodeDB := db.WithPrefix([]byte(nodeName))
	err := node.DeleteSyncedHeight(nodeDB)
	if err != nil {
		return errors.Wrap(err, "failed to delete synced height")
	}
	if err := node.DeletePendingTxs(nodeDB); err != nil {
		return errors.Wrap(err, "failed to delete pending txs")
	}
	if err := node.DeleteProcessedMsgs(nodeDB); err != nil {
		return errors.Wrap(err, "failed to delete processed msgs")
	}
	fmt.Printf("reset height to 0 for node %s\n", string(nodeDB.GetPrefix()))
	return nil
}

func Migration016(db types.DB) error {
	DAHostName := "da_host"
	DACelestiaName := "da_celestia"

	// move all data from da_host and da_celestia to da
	daDB := db.WithPrefix([]byte(types.DAName))
	for _, dbName := range []string{DAHostName, DACelestiaName} {
		nodeDB := db.WithPrefix([]byte(dbName))

		err := nodeDB.Iterate(nil, nil, func(key, value []byte) (bool, error) {
			err := daDB.Set(key, value)
			if err != nil {
				return true, errors.Wrap(err, "failed to set data to DA")
			}

			err = nodeDB.Delete(key)
			if err != nil {
				return true, errors.Wrap(err, fmt.Sprintf("failed to delete data from %s", dbName))
			}
			return false, nil
		})
		if err != nil {
			return err
		}
	}

	// change the last processed block height to synced height
	for _, nodeName := range []string{
		types.HostName,
		types.ChildName,
		types.BatchName,
		types.DAName,
	} {
		nodeDB := db.WithPrefix([]byte(nodeName))

		value, err := nodeDB.Get(nodetypes.LastProcessedBlockHeightKey)
		if err == nil {
			err = nodeDB.Set(nodetypes.SyncedHeightKey, value)
			if err != nil {
				return errors.Wrap(err, "failed to set synced height")
			}
		}
	}

	addressIndexMap := make(map[string]uint64)
	childDB := db.WithPrefix([]byte(types.ChildName))
	return childDB.Iterate(dbtypes.AppendSplitter(executortypes.WithdrawalPrefix), nil, func(key, value []byte) (bool, error) {
		if len(key) == len(executortypes.WithdrawalPrefix)+1+8 {
			data := executortypes.WithdrawalData{}
			err := data.Unmarshal(value)
			if err != nil {
				return true, err
			}
			addressIndexMap[data.To]++

			dataWithIndex := executortypes.NewWithdrawalDataWithIndex(data, addressIndexMap[data.To])
			dataBz, err := dataWithIndex.Marshal()
			if err != nil {
				return true, err
			}

			err = childDB.Set(executortypes.PrefixedWithdrawalSequence(data.Sequence), dataBz)
			if err != nil {
				return true, err
			}

			err = childDB.Set(executortypes.PrefixedWithdrawalAddressIndex(data.To, addressIndexMap[data.To]), dbtypes.FromUint64(data.Sequence))
			if err != nil {
				return true, err
			}
		}

		err := childDB.Delete(key)
		if err != nil {
			return true, err
		}
		return false, nil
	})
}
