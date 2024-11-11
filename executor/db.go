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
		types.DAHostName,
		types.DACelestiaName,
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
		nodeName != types.DAHostName &&
		nodeName != types.DACelestiaName {
		return errors.New("unknown node name")
	}
	nodeDB := db.WithPrefix([]byte(nodeName))
	err := node.DeleteSyncInfo(nodeDB)
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

func Migration015(db types.DB) error {
	nodeDB := db.WithPrefix([]byte(types.ChildName))
	addressIndexMap := make(map[string]uint64)
	return nodeDB.PrefixedIterate(executortypes.WithdrawalKey, nil, func(key, value []byte) (bool, error) {
		if len(key) != len(executortypes.WithdrawalKey)+1+8 {
			return false, nil
		}

		sequence := dbtypes.ToUint64Key(key[len(key)-8:])
		var data executortypes.WithdrawalData
		err := json.Unmarshal(value, &data)
		if err != nil {
			return true, err
		}
		addressIndexMap[data.To]++
		err = nodeDB.Set(executortypes.PrefixedWithdrawalKeyAddressIndex(data.To, addressIndexMap[data.To]), dbtypes.FromUint64(sequence))
		if err != nil {
			return true, err
		}
		return false, nil
	})
}
