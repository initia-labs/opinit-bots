package types

import (
	"errors"

	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
)

type Config struct {
	HostNode  HostConfig           `json:"host_node"`
	ChildNode nodetypes.NodeConfig `json:"child_node"`
	Version   uint8                `json:"version"`
	Address   string               `json:"address"`
}

type HostConfig struct {
	nodetypes.NodeConfig
	RelayOracle bool `json:"relay_oracle"`
}

func DefaultConfig() *Config {
	return &Config{
		HostNode: HostConfig{nodetypes.NodeConfig{
			RPC:      "tcp://localhost:26657",
			ChainID:  "localhost",
			Mnemonic: "",
			GasPrice: "0.15uinit",
		}, false},
		ChildNode: nodetypes.NodeConfig{
			RPC:      "tcp://localhost:27657",
			ChainID:  "l2",
			Mnemonic: "",
			GasPrice: "",
		},
		Version: 1,
		Address: "127.0.0.1:3000",
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
		return errors.New("version is required")
	}

	if cfg.Address == "" {
		return errors.New("address is required")
	}

	return nil
}
