package types

import (
	dbtypes "github.com/initia-labs/opinit-bots/db/types"
)

var (
	// Keys
	PendingTxsKey    = []byte("pending_txs")
	ProcessedMsgsKey = []byte("processed_msgs")
)

func PrefixedPendingTx(sequence uint64) []byte {
	return append(append(PendingTxsKey, dbtypes.Splitter), dbtypes.FromUint64Key(sequence)...)
}

func PrefixedProcessedMsgs(timestamp uint64) []byte {
	return append(append(ProcessedMsgsKey, dbtypes.Splitter), dbtypes.FromUint64Key(timestamp)...)
}
