package types

import (
	"time"
)

var (
	HostNodeName  = "host"
	ChildNodeName = "child"
)

const POLLING_INTERVAL = 1 * time.Second
const MSG_QUEUE_SIZE = 20
const MAX_PENDING_TXS = 5
const GAS_ADJUSTMENT = 1.5
const KEY_NAME = "key"
const TIMEOUT_HEIGHT = 100
