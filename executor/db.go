package executor

import (
	"encoding/json"
	"fmt"

	dbtypes "github.com/initia-labs/opinit-bots/db/types"
	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	"github.com/initia-labs/opinit-bots/node"
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
			return err
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
		return err
	}
	if err := node.DeletePendingTxs(nodeDB); err != nil {
		return err
	}
	if err := node.DeleteProcessedMsgs(nodeDB); err != nil {
		return err
	}
	fmt.Printf("reset height to 0 for node %s\n", string(nodeDB.GetPrefix()))
	return nil
}

func Migration016(db types.DB) error {
	DAHostName := "da_host"
	DACelestiaName := "da_celestia"

	daDB := db.WithPrefix([]byte(types.DAName))
	for _, dbName := range []string{DAHostName, DACelestiaName} {
		nodeDB := db.WithPrefix([]byte(dbName))

		err := nodeDB.PrefixedIterate(nil, nil, func(key, value []byte) (bool, error) {
			err := daDB.Set(key, value)
			if err != nil {
				return true, err
			}

			err = nodeDB.Delete(key)
			if err != nil {
				return true, err
			}
			return false, nil
		})
		if err != nil {
			return err
		}
	}

	addressIndexMap := make(map[string]uint64)

	childDB := db.WithPrefix([]byte(types.ChildName))
	return childDB.PrefixedIterate(dbtypes.AppendSplitter(executortypes.WithdrawalPrefix), nil, func(key, value []byte) (bool, error) {
		if len(key) == len(executortypes.WithdrawalPrefix)+1+8 {
			var data executortypes.WithdrawalData
			err := json.Unmarshal(value, &data)
			if err != nil {
				return true, err
			}
			addressIndexMap[data.To]++

			dataWithIndex := executortypes.WithdrawalDataWithIndex{
				Withdrawal: data,
				Index:      addressIndexMap[data.To],
			}

			dataBz, err := json.Marshal(dataWithIndex)
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
