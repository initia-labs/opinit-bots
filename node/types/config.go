package types

import (
	"fmt"

	btypes "github.com/initia-labs/opinit-bots-go/node/broadcaster/types"
)

type BlockProcessType uint8

const (
	PROCESS_TYPE_DEFAULT BlockProcessType = iota
	PROCESS_TYPE_RAW
	PROCESS_TYPE_ONLY_BROADCAST
)

type NodeConfig struct {
	RPC string

	// BlockProcessType is the type of block process.
	ProcessType BlockProcessType

	// You can leave it empty, then the bot will skip the transaction submission.
	BroadcasterConfig *btypes.BroadcasterConfig
}

func (nc NodeConfig) Validate() error {
	if nc.RPC == "" {
		return fmt.Errorf("rpc is empty")
	}

	if nc.ProcessType > PROCESS_TYPE_ONLY_BROADCAST {
		return fmt.Errorf("invalid process type")
	}

	// Validated in broadcaster
	//
	// if nc.BroadcasterConfig != nil {
	// 	if err := nc.BroadcasterConfig.Validate(); err != nil {
	// 		return err
	// 	}
	// }

	return nil
}
