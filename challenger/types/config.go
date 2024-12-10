package types

import (
	"errors"

	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	servertypes "github.com/initia-labs/opinit-bots/server/types"
)

type NodeConfig struct {
	ChainID      string `json:"chain_id"`
	Bech32Prefix string `json:"bech32_prefix"`
	RPCAddress   string `json:"rpc_address"`
}

func (nc NodeConfig) Validate() error {
	if nc.ChainID == "" {
		return errors.New("chain ID is required")
	}
	if nc.Bech32Prefix == "" {
		return errors.New("bech32 prefix is required")
	}
	if nc.RPCAddress == "" {
		return errors.New("RPC address is required")
	}
	return nil
}

type Config struct {
	// Version is the version used to build output root.
	Version uint8 `json:"version"`

	// Server is the configuration for the server.
	Server servertypes.ServerConfig `json:"server"`

	// L1Node is the configuration for the l1 node.
	L1Node NodeConfig `json:"l1_node"`
	// L2Node is the configuration for the l2 node.
	L2Node NodeConfig `json:"l2_node"`

	// DisableAutoSetL1Height is the flag to disable the automatic setting of the l1 height.
	// If it is false, it will finds the optimal height and sets l1_start_height automatically
	// from l2 start height and l1_start_height is ignored.
	// It can be useful when you don't want to use TxSearch.
	DisableAutoSetL1Height bool `json:"disable_auto_set_l1_height"`
	// L1StartHeight is the height to start the l1 node.
	L1StartHeight int64 `json:"l1_start_height"`
	// L2StartHeight is the height to start the l2 node. If it is 0, it will start from the latest height.
	// If the latest height stored in the db is not 0, this config is ignored.
	// L2 starts from the last submitted output l2 block number + 1 before L2StartHeight.
	// L1 starts from the block number of the output tx + 1
	L2StartHeight int64 `json:"l2_start_height"`
}

func DefaultConfig() *Config {
	return &Config{
		Version: 1,

		Server: servertypes.ServerConfig{
			Address:      "localhost:3001",
			AllowOrigins: "*",
			AllowHeaders: "Origin, Content-Type, Accept",
			AllowMethods: "GET",
		},

		L1Node: NodeConfig{
			ChainID:      "testnet-l1-1",
			Bech32Prefix: "init",
			RPCAddress:   "tcp://localhost:26657",
		},

		L2Node: NodeConfig{
			ChainID:      "testnet-l2-1",
			Bech32Prefix: "init",
			RPCAddress:   "tcp://localhost:27657",
		},
		DisableAutoSetL1Height: false,
		L1StartHeight:          0,
		L2StartHeight:          0,
	}
}

func (cfg Config) Validate() error {
	if cfg.Version == 0 {
		return errors.New("version is required")
	}

	if cfg.Version != 1 {
		return errors.New("only version 1 is supported")
	}

	if err := cfg.Server.Validate(); err != nil {
		return err
	}

	if err := cfg.L1Node.Validate(); err != nil {
		return err
	}

	if err := cfg.L2Node.Validate(); err != nil {
		return err
	}

	if cfg.L1StartHeight < 0 {
		return errors.New("l1 start height must be greater than or equal to 0")
	}

	if cfg.L2StartHeight < 0 {
		return errors.New("l2 start height must be greater than or equal to 0")
	}
	return nil
}

func (cfg Config) L1NodeConfig() nodetypes.NodeConfig {
	nc := nodetypes.NodeConfig{
		RPC:          cfg.L1Node.RPCAddress,
		ProcessType:  nodetypes.PROCESS_TYPE_DEFAULT,
		Bech32Prefix: cfg.L1Node.Bech32Prefix,
	}
	return nc
}

func (cfg Config) L2NodeConfig() nodetypes.NodeConfig {
	nc := nodetypes.NodeConfig{
		RPC:          cfg.L2Node.RPCAddress,
		ProcessType:  nodetypes.PROCESS_TYPE_DEFAULT,
		Bech32Prefix: cfg.L2Node.Bech32Prefix,
	}
	return nc
}
