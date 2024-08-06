# Merkle tree implementation
A data structure for proving withdrawals in `OPInit`. 

## Working Tree
When initializing working tree, if this is not the first time making it and there is no working tree of the previous version, it will cause `panic`. 
The working tree does not have all nodes in memory. Since the leaf nodes used when creating a tree have a `sequence`, a tree can be created deterministically and it does not need to sort leaf nodes. Thus, it only has one sibling node for each height of the last tree required for the operation. The node for which the operation is completed is stored in the DB using treeIndex, height, and nodeIndex for later querying. The leaf index of a tree increases sequentially as leaves are added.
``` go
func GetNodeKey(treeIndex uint64, height uint8, nodeIndex uint64) []byte {
	data := make([]byte, 17)
	binary.BigEndian.PutUint64(data, treeIndex)
	data[8] = height
	binary.BigEndian.PutUint64(data[9:], nodeIndex)
	return data
}
```

## Node generation function
The node generation function must be deterministic regardless of how the child nodes are sorted. The default functions are as follows:
```go
func GenerateNodeHash(a, b []byte) [32]byte {
	var data [32]byte
	switch bytes.Compare(a, b) {
	case 0, 1: // equal or greater
		data = sha3.Sum256(append(b, a...))
	case -1: // less
		data = sha3.Sum256(append(a, b...))
	}
	return data
}
```

## Finalizing working tree
When finalizing the current working tree, the remaining leaf nodes are filled in with the last leaf node to make the tree a complete binary tree. The finalized tree is stored with `startLeafIndex` as the key. The current working tree will be marked as done, and the next tree will start anew from leaf index 0.

## Withdrawal proofs
To query the data of a leaf node and its corresponding proof, we need to find the corresponding finalized tree stored there. First, it finds the last finalized tree index that is less than or equal to the index it want to find, and then it can specify the leaf node index of the tree with (querying index - start leaf index). The sibling nodes from this leaf node to the root are provided as merkle proofs.