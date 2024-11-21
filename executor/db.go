package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"math/bits"
	"time"

	dbtypes "github.com/initia-labs/opinit-bots/db/types"
	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	merkletypes "github.com/initia-labs/opinit-bots/merkle/types"
	"github.com/initia-labs/opinit-bots/node"
	"github.com/initia-labs/opinit-bots/node/rpcclient"
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

func Migration0191(db types.DB) error {
	nodeDB := db.WithPrefix([]byte(types.ChildName))
	merkleDB := nodeDB.WithPrefix([]byte(types.MerkleName))

	err := merkleDB.PrefixedIterate(merkletypes.FinalizedTreeKey, nil, func(key, value []byte) (bool, error) {
		var tree merkletypes.FinalizedTreeInfo
		err := json.Unmarshal(value, &tree)
		if err != nil {
			return true, err
		}

		err = merkleDB.Delete(key)
		if err != nil {
			return true, err
		}
		fmt.Printf("delete finalized tree index: %d, start leaf index: %d, leaf count: %d\n", tree.TreeIndex, tree.StartLeafIndex, tree.LeafCount)
		return false, nil
	})
	if err != nil {
		return err
	}

	nextSequence := uint64(1)
	changeWorkingTree := false
	err = merkleDB.PrefixedIterate(merkletypes.WorkingTreeKey, nil, func(key, value []byte) (bool, error) {
		if len(key) != len(merkletypes.WorkingTreeKey)+1+8 {
			return false, nil
		}

		version := dbtypes.ToUint64Key(key[len(key)-8:])

		var workingTree merkletypes.TreeInfo
		err := json.Unmarshal(value, &workingTree)
		if err != nil {
			return true, err
		}

		if nextSequence != workingTree.StartLeafIndex {
			changeWorkingTree = true
		}

		if changeWorkingTree {
			workingTree.StartLeafIndex = nextSequence
			workingTreeBz, err := json.Marshal(workingTree)
			if err != nil {
				return true, err
			}
			err = merkleDB.Set(key, workingTreeBz)
			if err != nil {
				return true, err
			}
		}

		if workingTree.Done && workingTree.LeafCount != 0 {
			data, err := json.Marshal(executortypes.TreeExtraData{
				BlockNumber: types.MustUint64ToInt64(version),
			})
			if err != nil {
				return true, err
			}

			treeHeight := types.MustIntToUint8(bits.Len64(workingTree.LeafCount - 1))
			if workingTree.LeafCount <= 1 {
				treeHeight = uint8(workingTree.LeafCount) //nolint
			}

			treeRootHash := workingTree.LastSiblings[treeHeight]
			finalizedTreeInfo := merkletypes.FinalizedTreeInfo{
				TreeIndex:      workingTree.Index,
				TreeHeight:     treeHeight,
				Root:           treeRootHash,
				StartLeafIndex: workingTree.StartLeafIndex,
				LeafCount:      workingTree.LeafCount,
				ExtraData:      data,
			}

			finalizedTreeBz, err := json.Marshal(finalizedTreeInfo)
			if err != nil {
				return true, err
			}

			err = merkleDB.Set(finalizedTreeInfo.Key(), finalizedTreeBz)
			if err != nil {
				return true, err
			}

			fmt.Printf("finalized tree index: %d, start leaf index: %d, leaf count: %d, block height: %d\n", finalizedTreeInfo.TreeIndex, finalizedTreeInfo.StartLeafIndex, finalizedTreeInfo.LeafCount, version)
			nextSequence = workingTree.StartLeafIndex + workingTree.LeafCount
		}
		return false, nil
	})
	if err != nil {
		return err
	}
	return nil
}

func Migration0192(ctx context.Context, db types.DB, rpcClient *rpcclient.RPCClient) error {
	nodeDB := db.WithPrefix([]byte(types.ChildName))
	merkleDB := nodeDB.WithPrefix([]byte(types.MerkleName))

	timer := time.NewTicker(types.PollingInterval(ctx))
	defer timer.Stop()

	return merkleDB.PrefixedIterate(merkletypes.FinalizedTreeKey, nil, func(key, value []byte) (bool, error) {
		var tree merkletypes.FinalizedTreeInfo
		err := json.Unmarshal(value, &tree)
		if err != nil {
			return true, err
		}

		var extraData executortypes.TreeExtraData
		err = json.Unmarshal(tree.ExtraData, &extraData)
		if err != nil {
			return true, err
		}

		if extraData.BlockHash != nil {
			return false, nil
		}

		for {
			select {
			case <-ctx.Done():
				return true, ctx.Err()
			case <-timer.C:
			}
			height := extraData.BlockNumber + 1
			header, err := rpcClient.Header(ctx, &height)
			if err != nil {
				continue
			}

			extraData.BlockHash = header.Header.LastBlockID.Hash
			break
		}

		tree.ExtraData, err = json.Marshal(extraData)
		if err != nil {
			return true, err
		}
		treeBz, err := json.Marshal(tree)
		if err != nil {
			return true, err
		}
		err = merkleDB.Set(key, treeBz)
		if err != nil {
			return true, err
		}
		fmt.Printf("finalized tree index: %d, start leaf index: %d, leaf count: %d, block height: %d, block hash: %X\n", tree.TreeIndex, tree.StartLeafIndex, tree.LeafCount, extraData.BlockNumber, extraData.BlockHash)
		return false, nil
	})
}
