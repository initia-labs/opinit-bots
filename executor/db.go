package executor

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/bits"
	"time"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	dbtypes "github.com/initia-labs/opinit-bots/db/types"
	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	merkletypes "github.com/initia-labs/opinit-bots/merkle/types"
	"github.com/initia-labs/opinit-bots/node"
	"github.com/initia-labs/opinit-bots/node/rpcclient"
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

func Migration015(db types.DB) error {
	nodeDB := db.WithPrefix([]byte(types.ChildName))
	addressIndexMap := make(map[string]uint64)
	return nodeDB.Iterate(executortypes.WithdrawalPrefix, nil, func(key, value []byte) (bool, error) {
		if len(key) != len(executortypes.WithdrawalPrefix)+1+8 {
			return false, nil
		}

		sequence := dbtypes.ToUint64Key(key[len(key)-8:])
		var data executortypes.WithdrawalData
		err := json.Unmarshal(value, &data)
		if err != nil {
			return true, err
		}
		addressIndexMap[data.To]++
		err = nodeDB.Set(executortypes.PrefixedWithdrawalAddressSequence(data.To, addressIndexMap[data.To]), dbtypes.FromUint64(sequence))
		if err != nil {
			return true, err
		}
		return false, nil
	})
}

func Migration019_1(db types.DB) error {
	nodeDB := db.WithPrefix([]byte(types.ChildName))
	merkleDB := nodeDB.WithPrefix([]byte(types.MerkleName))

	err := merkleDB.Iterate(merkletypes.FinalizedTreePrefix, nil, func(key, value []byte) (bool, error) {
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
	err = merkleDB.Iterate(merkletypes.WorkingTreePrefix, nil, func(key, value []byte) (bool, error) {
		if len(key) != len(merkletypes.WorkingTreePrefix)+1+8 {
			return true, fmt.Errorf("unexpected working tree key; expected: %d; got: %d", len(merkletypes.WorkingTreePrefix)+1+8, len(key))
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

func Migration019_2(ctx types.Context, db types.DB, rpcClient *rpcclient.RPCClient) error {
	nodeDB := db.WithPrefix([]byte(types.ChildName))
	merkleDB := nodeDB.WithPrefix([]byte(types.MerkleName))

	timer := time.NewTicker(ctx.PollingInterval())
	defer timer.Stop()

	return merkleDB.Iterate(merkletypes.FinalizedTreePrefix, nil, func(key, value []byte) (bool, error) {
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
				fmt.Printf("failed to get header for block height: %d; %s\n", height, err.Error())
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
		outputRoot := ophosttypes.GenerateOutputRoot(1, tree.Root, extraData.BlockHash)
		outputRootStr := base64.StdEncoding.EncodeToString(outputRoot[:])

		fmt.Printf("finalized tree index: %d, start leaf index: %d, leaf count: %d, block height: %d, block hash: %X, outputRoot: %s\n", tree.TreeIndex, tree.StartLeafIndex, tree.LeafCount, extraData.BlockNumber, extraData.BlockHash, outputRootStr)
		return false, nil
	})
}

func Migration0110(db types.DB) error {
	nodeDB := db.WithPrefix([]byte(types.ChildName))
	err := nodeDB.Iterate(executortypes.WithdrawalPrefix, nil, func(key, value []byte) (bool, error) {
		// pass PrefixedWithdrawalKey ( WithdrawalKey / Sequence )
		// we only delete PrefixedWithdrawalKeyAddressIndex ( WithdrawalKey / Address / Sequence )
		if len(key) == len(executortypes.WithdrawalPrefix)+1+8 {
			return false, nil
		}
		err := nodeDB.Delete(key)
		if err != nil {
			return true, err
		}
		return false, nil
	})
	if err != nil {
		return err
	}

	return nodeDB.Iterate(executortypes.WithdrawalPrefix, nil, func(key, value []byte) (bool, error) {
		sequence := dbtypes.ToUint64Key(key[len(key)-8:])
		var data executortypes.WithdrawalData
		err := json.Unmarshal(value, &data)
		if err != nil {
			return true, err
		}
		err = nodeDB.Set(executortypes.PrefixedWithdrawalAddressSequence(data.To, sequence), dbtypes.FromUint64(sequence))
		if err != nil {
			return true, err
		}
		return false, nil
	})
}

func Migration0111(db types.DB) error {
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

		value, err := nodeDB.Get([]byte("last_processed_block_height"))
		if err == nil {
			err = nodeDB.Set(nodetypes.SyncedHeightKey, value)
			if err != nil {
				return errors.Wrap(err, "failed to set synced height")
			}
		}
	}

	childDB := db.WithPrefix([]byte(types.ChildName))
	return childDB.Iterate(merkletypes.WorkingTreePrefix, nil, func(key, value []byte) (bool, error) {
		version, err := merkletypes.ParseWorkingTreeKey(key)
		if err != nil {
			return true, errors.Wrap(err, "failed to parse working tree key")
		}

		var legacyTree merkletypes.LegacyTreeInfo
		err = json.Unmarshal(value, &legacyTree)
		if err != nil {
			return true, errors.Wrap(err, "failed to unmarshal tree info")
		}

		tree := legacyTree.Migrate(version)
		treeBz, err := tree.Marshal()
		if err != nil {
			return true, errors.Wrap(err, "failed to marshal tree info")
		}

		err = childDB.Set(key, treeBz)
		if err != nil {
			return true, errors.Wrap(err, "failed to set tree info")
		}
		return false, nil
	})
}
