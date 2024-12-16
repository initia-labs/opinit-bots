package merkle

import (
	"encoding/json"
	"fmt"

	dbtypes "github.com/initia-labs/opinit-bots/db/types"
	merkletypes "github.com/initia-labs/opinit-bots/merkle/types"
	types "github.com/initia-labs/opinit-bots/types"

	"github.com/pkg/errors"
)

// DeleteFutureFinalizedTrees deletes all finalized trees with sequence number greater than or equal to fromSequence.
func DeleteFutureFinalizedTrees(db types.DB, fromSequence uint64) error {
	return db.Iterate(dbtypes.AppendSplitter(merkletypes.FinalizedTreePrefix), merkletypes.PrefixedFinalizedTreeKey(fromSequence), func(key, _ []byte) (bool, error) {
		err := db.Delete(key)
		if err != nil {
			return true, err
		}
		return false, nil
	})
}

// DeleteFutureWorkingTrees deletes all working trees with version greater than or equal to fromVersion.
func DeleteFutureWorkingTrees(db types.DB, fromVersion uint64) error {
	return db.Iterate(dbtypes.AppendSplitter(merkletypes.WorkingTreePrefix), merkletypes.PrefixedWorkingTreeKey(fromVersion), func(key, _ []byte) (bool, error) {
		err := db.Delete(key)
		if err != nil {
			return true, err
		}
		return false, nil
	})
}

// GetWorkingTree returns the working tree with the given version.
func GetWorkingTree(db types.BasicDB, version uint64) (merkletypes.TreeInfo, error) {
	data, err := db.Get(merkletypes.PrefixedWorkingTreeKey(version))
	if err != nil {
		return merkletypes.TreeInfo{}, err
	}

	var workingTree merkletypes.TreeInfo
	err = json.Unmarshal(data, &workingTree)
	return workingTree, err
}

// SaveWorkingTree saves the working tree to the db.
func SaveWorkingTree(db types.BasicDB, workingTree merkletypes.TreeInfo) error {
	value, err := workingTree.Value()
	if err != nil {
		return err
	}
	return db.Set(workingTree.Key(), value)
}

// GetFinalizedTree returns the finalized tree with the given start leaf index.
func GetFinalizedTree(db types.BasicDB, startLeafIndex uint64) (merkletypes.FinalizedTreeInfo, error) {
	data, err := db.Get(merkletypes.PrefixedFinalizedTreeKey(startLeafIndex))
	if err != nil {
		return merkletypes.FinalizedTreeInfo{}, err
	}

	var finalizedTree merkletypes.FinalizedTreeInfo
	err = json.Unmarshal(data, &finalizedTree)
	return finalizedTree, err
}

// SaveFinalizedTree saves the finalized tree to the db.
func SaveFinalizedTree(db types.BasicDB, finalizedTree merkletypes.FinalizedTreeInfo) error {
	value, err := finalizedTree.Value()
	if err != nil {
		return err
	}
	return db.Set(finalizedTree.Key(), value)
}

// SaveNodes saves the nodes to the db.
func SaveNodes(db types.BasicDB, nodes ...merkletypes.Node) error {
	for _, node := range nodes {
		err := db.Set(node.Key(), node.Value())
		if err != nil {
			return err
		}
	}
	return nil
}

// GetNodeBytes returns the node with the given tree index, height, and local node index.
func GetNodeBytes(db types.BasicDB, treeIndex uint64, height uint8, localNodeIndex uint64) ([]byte, error) {
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
		// In `FinalizeWorkingTree`, we ensure that the leaf count of the tree is always a power of two by filling the leaves as needed.
		// This ensures that there is always a sibling for each leaf node.
		siblingIndex := localNodeIndex ^ 1 // flip the last bit to find the sibling
		sibling, err := GetNodeBytes(db, treeInfo.TreeIndex, height, siblingIndex)
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
