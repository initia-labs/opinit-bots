package types

import "time"

const (
	HostNodeName  = "host"
	ChildNodeName = "child"
	MerkleName    = "merkle"

	OutputInterval       = 1 * time.Minute
	OutputIntervalHeight = 60
)

type WithdrawalHash [32]byte
