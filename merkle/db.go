package merkle

import (
	"encoding/json"
	"fmt"

	dbtypes "github.com/initia-labs/opinit-bots/db/types"
	merkletypes "github.com/initia-labs/opinit-bots/merkle/types"
	types "github.com/initia-labs/opinit-bots/types"

	"github.com/pkg/errors"
)

func DeleteFutureFinalizedTrees(db types.DB, fromSequence uint64) error {
	return db.Iterate(dbtypes.AppendSplitter(merkletypes.FinalizedTreePrefix), nil, func(key, _ []byte) (bool, error) {
		sequence, err := merkletypes.ParseFinalizedTreeKey(key)
		if err != nil {
			return true, err
		}

		if sequence >= fromSequence {
			err := db.Delete(key)
			if err != nil {
				return true, err
			}
		}
		return false, nil
	})
}

func DeleteFutureWorkingTrees(db types.DB, fromVersion uint64) error {
	return db.Iterate(dbtypes.AppendSplitter(merkletypes.WorkingTreePrefix), nil, func(key, _ []byte) (bool, error) {
		version, err := merkletypes.ParseWorkingTreeKey(key)
		if err != nil {
			return true, err
		}

		if version >= fromVersion {
			err := db.Delete(key)
			if err != nil {
				return true, err
			}
		}
		return false, nil
	})
}

func GetWorkingTree(db types.BasicDB, version uint64) (merkletypes.TreeInfo, error) {
	data, err := db.Get(merkletypes.PrefixedWorkingTreeKey(version))
	if err != nil {
		return merkletypes.TreeInfo{}, err
	}

	var workingTree merkletypes.TreeInfo
	err = json.Unmarshal(data, &workingTree)
	return workingTree, err
}

func SaveWorkingTree(db types.BasicDB, workingTree merkletypes.TreeInfo) error {
	value, err := workingTree.Value()
	if err != nil {
		return err
	}
	return db.Set(workingTree.Key(), value)
}

func SaveFinalizedTree(db types.BasicDB, finalizedTree merkletypes.FinalizedTreeInfo) error {
	value, err := json.Marshal(finalizedTree)
	if err != nil {
		return err
	}
	return db.Set(finalizedTree.Key(), value)
}

func SaveNodes(db types.BasicDB, nodes ...merkletypes.Node) error {
	for _, node := range nodes {
		err := db.Set(node.Key(), node.Value())
		if err != nil {
			return err
		}
	}
	return nil
}

func GetNode(db types.BasicDB, treeIndex uint64, height uint8, localNodeIndex uint64) ([]byte, error) {
	return db.Get(merkletypes.PrefixedNodeKey(treeIndex, height, localNodeIndex))
}

// GetProofs returns the proofs for the leaf with the given index.
func GetProofs(db types.DB, leafIndex uint64) (proofs [][]byte, treeIndex uint64, rootData []byte, extraData []byte, err error) {
	_, value, err := db.SeekPrevInclusiveKey(merkletypes.FinalizedTreePrefix, merkletypes.PrefixedFinalizedTreeKey(leafIndex))
	if errors.Is(err, dbtypes.ErrNotFound) {
		return nil, 0, nil, nil, merkletypes.ErrUnfinalizedTree
	} else if err != nil {
		return nil, 0, nil, nil, err
	}

	var treeInfo merkletypes.FinalizedTreeInfo
	if err := json.Unmarshal(value, &treeInfo); err != nil {
		return nil, 0, nil, nil, err
	}

	// Check if the leaf index is in the tree
	if leafIndex < treeInfo.StartLeafIndex {
		return nil, 0, nil, nil, fmt.Errorf("leaf (`%d`) is not found in tree (`%d`)", leafIndex, treeInfo.TreeIndex)
	} else if leafIndex-treeInfo.StartLeafIndex >= treeInfo.LeafCount {
		return nil, 0, nil, nil, merkletypes.ErrUnfinalizedTree
	}

	height := uint8(0)
	localNodeIndex := leafIndex - treeInfo.StartLeafIndex
	for height < treeInfo.TreeHeight {
		siblingIndex := localNodeIndex ^ 1 // flip the last bit to find the sibling
		sibling, err := GetNode(db, treeInfo.TreeIndex, height, siblingIndex)
		if err != nil {
			return nil, 0, nil, nil, errors.Wrap(err, "failed to get sibling node from db")
		}

		// append the sibling to the proofs
		proofs = append(proofs, sibling)

		// update iteration variables
		height++
		localNodeIndex = localNodeIndex / 2
	}

	return proofs, treeInfo.TreeIndex, treeInfo.Root, treeInfo.ExtraData, nil
}
