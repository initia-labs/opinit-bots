package types

import (
	"time"
)

const POLLING_INTERVAL = 1 * time.Second
const MSG_QUEUE_SIZE = 20
const MAX_PENDING_TXS = 5
const GAS_ADJUSTMENT = 1.5
const TX_TIMEOUT = 30 * time.Second

type BlockProcessType uint8

const (
	PROCESS_TYPE_DEFAULT BlockProcessType = iota
	PROCESS_TYPE_RAW
	PROCESS_TYPE_ONLY_BROADCAST
)
