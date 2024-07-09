package types

import (
	"fmt"
	"time"
)

var (
	HostNodeName  = "host"
	ChildNodeName = "child"

	LastProcessedBlockHeight = []byte("last_processed_block_height")
	PrefixPendingTxs         = []byte("pending_txs")
	ProcessedDataKey         = []byte("processed_data")
)

const POLLING_INTERVAL = 1 * time.Second
const MSG_QUEUE_SIZE = 20
const MAX_PENDING_TXS = 5
const GAS_ADJUSTMENT = 1.5
const KEY_NAME = "key"
const TIMEOUT_HEIGHT = 100

const PER_MSG_GAS_LIMIT = 500_000

func PrefixedPendingTx(sequence uint64) []byte {
	return []byte(fmt.Sprintf("%s/%d", string(PrefixPendingTxs), sequence))
}
