package executor

import (
	"fmt"

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
