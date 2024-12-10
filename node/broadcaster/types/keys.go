package types

import (
	dbtypes "github.com/initia-labs/opinit-bots/db/types"
)

var (
	// Keys
	PendingTxsPrefix    = []byte("pending_txs")
	ProcessedMsgsPrefix = []byte("processed_msgs")
)

func prefixedPendingTx(timestamp uint64) []byte {
	return dbtypes.GenerateKey([][]byte{
		PendingTxsPrefix,
		dbtypes.FromUint64Key(timestamp),
	})
}

func prefixedProcessedMsgs(timestamp uint64) []byte {
	return dbtypes.GenerateKey([][]byte{
		ProcessedMsgsPrefix,
		dbtypes.FromUint64Key(timestamp),
	})
}
