package types

import (
	"errors"

	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
)

type Config struct {
	Version   uint8                `json:"version"`
	HostNode  nodetypes.NodeConfig `json:"host_node"`
	ChildNode nodetypes.NodeConfig `json:"child_node"`
	Batch     BatchConfig          `json:"batch"`
}

func DefaultConfig() *Config {
	return &Config{
		Version: 1,
		HostNode: nodetypes.NodeConfig{
			RPC:     "tcp://localhost:26657",
			ChainID: "localhost",
		},
		ChildNode: nodetypes.NodeConfig{
			RPC:      "tcp://localhost:27657",
			ChainID:  "l2",
			Mnemonic: "",
			GasPrice: "0.15umin",
		},
	}
}

func (cfg Config) Validate() error {
	if cfg.HostNode.ChainID == "" {
		return errors.New("L1 chain ID is required")
	}

	if cfg.HostNode.RPC == "" {
		return errors.New("L1 RPC URL is required")
	}

	if cfg.ChildNode.ChainID == "" {
		return errors.New("L2 chain ID is required")
	}

	if cfg.ChildNode.RPC == "" {
		return errors.New("L2 RPC URL is required")
	}

	if cfg.Version == 0 {
		return errors.New("Bridge ID is required")
	}
	return nil
}

type BatchConfig struct {
	DANode       nodetypes.NodeConfig `json:"da_node"`
	MaxBatchSize int64                `json:"max_batch_size"`
}
