package executor

import (
	"fmt"

	"github.com/initia-labs/opinit-bots/node"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"
)

func ResetHeights(db types.DB) error {
	dbs := []types.DB{
		db.WithPrefix([]byte(types.HostName)),
		db.WithPrefix([]byte(types.ChildName)),
		db.WithPrefix([]byte(types.BatchName)),
	}

	for _, db := range dbs {
		if err := node.DeleteSyncInfo(db); err != nil {
			return err
		}
		if err := node.DeletePendingTxs(db); err != nil {
			return err
		}
		if err := node.DeleteProcessedMsgs(db); err != nil {
			return err
		}
		fmt.Printf("reset height to 0 for node %s\n", string(db.GetPrefix()))
	}
	return nil
}

func ResetHeight(db types.DB, nodeName string) error {
	if nodeName != types.HostName &&
		nodeName != types.ChildName &&
		nodeName != types.BatchName {
		return errors.New("unknown node name")
	}
	nodeDB := db.WithPrefix([]byte(nodeName))
	err := node.DeleteSyncInfo(nodeDB)
	if err != nil {
		return err
	}
	if err := node.DeletePendingTxs(db); err != nil {
		return err
	}
	if err := node.DeleteProcessedMsgs(db); err != nil {
		return err
	}
	fmt.Printf("reset height to 0 for node %s\n", string(nodeDB.GetPrefix()))
	return nil
}
