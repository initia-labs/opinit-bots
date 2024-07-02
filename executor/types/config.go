package types

import (
	"errors"

	"github.com/initia-labs/opinit-bots-go/node"
)

type Config struct {
	HostNode  node.NodeConfig `json:"host_node"`
	ChildNode node.NodeConfig `json:"child_node"`
	BridgeId  int64           `json:"bridge_id"`
}

func DefaultConfig() *Config {
	return &Config{
		HostNode: node.NodeConfig{
			RPC:     "tcp://localhost:26657",
			ChainID: "localhost",
		},
		ChildNode: node.NodeConfig{
			RPC:      "tcp://localhost:27657",
			ChainID:  "l2",
			Mnemonic: "",
			GasPrice: "0.15umin",
		},
		BridgeId: 0,
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

	if cfg.BridgeId == 0 {
		return errors.New("Bridge ID is required")
	}

	return nil
}
