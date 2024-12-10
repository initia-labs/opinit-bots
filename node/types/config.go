package types

import (
	"fmt"

	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
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

	// Bech32Prefix is the Bech32 prefix of the chain.
	Bech32Prefix string

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

	if nc.Bech32Prefix == "" {
		return fmt.Errorf("bech32 prefix is empty")
	}
	return nil
}
