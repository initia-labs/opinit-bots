package types

import (
	"errors"

	nodetypes "github.com/initia-labs/opinit-bots/node/types"
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

	// ListenAddress is the address to listen for incoming requests.
	ListenAddress string `json:"listen_address"`

	// L1Node is the configuration for the l1 node.
	L1Node NodeConfig `json:"l1_node"`
	// L2Node is the configuration for the l2 node.
	L2Node NodeConfig `json:"l2_node"`

	// L2StartHeight is the height to start the l2 node. If it is 0, it will start from the latest height.
	// If the latest height stored in the db is not 0, this config is ignored.
	// L2 starts from the last submitted output l2 block number + 1 before L2StartHeight.
	// L1 starts from the block number of the output tx + 1
	L2StartHeight uint64 `json:"l2_start_height"`

	WarningThreshold float64 `json:"warning_threshold"`
}

func DefaultConfig() *Config {
	return &Config{
		Version:       1,
		ListenAddress: "localhost:3000",

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
		L2StartHeight: 0,

		WarningThreshold: 0.8,
	}
}

func (cfg Config) Validate() error {
	if cfg.Version == 0 {
		return errors.New("version is required")
	}

	if cfg.Version != 1 {
		return errors.New("only version 1 is supported")
	}

	if cfg.ListenAddress == "" {
		return errors.New("listen address is required")
	}

	if err := cfg.L1Node.Validate(); err != nil {
		return err
	}

	if err := cfg.L2Node.Validate(); err != nil {
		return err
	}
	return nil
}

func (cfg Config) L1NodeConfig(homePath string) nodetypes.NodeConfig {
	nc := nodetypes.NodeConfig{
		RPC:         cfg.L1Node.RPCAddress,
		ProcessType: nodetypes.PROCESS_TYPE_DEFAULT,
	}
	return nc
}

func (cfg Config) L2NodeConfig(homePath string) nodetypes.NodeConfig {
	nc := nodetypes.NodeConfig{
		RPC:         cfg.L2Node.RPCAddress,
		ProcessType: nodetypes.PROCESS_TYPE_DEFAULT,
	}
	return nc
}

type ChallengeConfig struct {
	WarningThreshold float64
}

func (cfg Config) ChallengeConfig() ChallengeConfig {
	return ChallengeConfig{
		WarningThreshold: cfg.WarningThreshold,
	}
}
