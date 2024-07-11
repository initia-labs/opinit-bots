package types

import (
	dbtypes "github.com/initia-labs/opinit-bots-go/db/types"
)

var (
	LastProcessedBlockHeightKey = []byte("last_processed_block_height")

	PendingTxsKey = []byte("pending_txs")
	// be used to iterate all pending txs
	LastPendingTxKey = append(PendingTxsKey, dbtypes.Splitter+1)

	ProcessedMsgsKey = []byte("processed_msgs")
	// be used to iterate all processed msgs
	LastProcessedMsgsKey = append(ProcessedMsgsKey, dbtypes.Splitter+1)
)

func PrefixedPendingTx(sequence uint64) []byte {
	return append(append(PendingTxsKey, dbtypes.Splitter), dbtypes.FromUint64Key(sequence)...)
}

func PrefixedProcessedMsgs(timestamp uint64) []byte {
	return append(append(ProcessedMsgsKey, dbtypes.Splitter), dbtypes.FromUint64Key(timestamp)...)
}
