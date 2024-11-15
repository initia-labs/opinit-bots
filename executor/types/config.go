package types

import (
	"errors"
	"time"

	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"

	servertypes "github.com/initia-labs/opinit-bots/server/types"
)

type NodeConfig struct {
	ChainID       string  `json:"chain_id"`
	Bech32Prefix  string  `json:"bech32_prefix"`
	RPCAddress    string  `json:"rpc_address"`
	GasPrice      string  `json:"gas_price"`
	GasAdjustment float64 `json:"gas_adjustment"`
	TxTimeout     int64   `json:"tx_timeout"` // seconds
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
	// DANode is the configuration for the data availability node.
	DANode NodeConfig `json:"da_node"`

	// BridgeExecutor is the key name in the keyring for the bridge executor,
	// which is used to relay initiate token bridge transaction from l1 to l2.
	//
	// If you don't want to use the bridge executor feature, you can leave it empty.
	BridgeExecutor string `json:"bridge_executor"`

	// DisableOutputSubmitter is the flag to disable the output submitter.
	// If it is true, the output submitter will not be started.
	DisableOutputSubmitter bool `json:"disable_output_submitter"`

	// DisableBatchSubmitter is the flag to disable the batch submitter.
	// If it is true, the batch submitter will not be started.
	DisableBatchSubmitter bool `json:"disable_batch_submitter"`

	// MaxChunks is the maximum number of chunks in a batch.
	MaxChunks int64 `json:"max_chunks"`
	// MaxChunkSize is the maximum size of a chunk in a batch.
	MaxChunkSize int64 `json:"max_chunk_size"`
	// MaxSubmissionTime is the maximum time to submit a batch.
	MaxSubmissionTime int64 `json:"max_submission_time"` // seconds

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
	// BatchStartHeight is the height to start the batch. If it is 0, it will start from the latest height.
	// If the latest height stored in the db is not 0, this config is ignored.
	BatchStartHeight int64 `json:"batch_start_height"`
}

func DefaultConfig() *Config {
	return &Config{
		Version: 1,

		Server: servertypes.ServerConfig{
			Address:      "localhost:3000",
			AllowOrigins: "*",
			AllowHeaders: "Origin, Content-Type, Accept",
			AllowMethods: "GET",
		},

		L1Node: NodeConfig{
			ChainID:       "testnet-l1-1",
			Bech32Prefix:  "init",
			RPCAddress:    "tcp://localhost:26657",
			GasPrice:      "0.15uinit",
			GasAdjustment: 1.5,
			TxTimeout:     60,
		},

		L2Node: NodeConfig{
			ChainID:       "testnet-l2-1",
			Bech32Prefix:  "init",
			RPCAddress:    "tcp://localhost:27657",
			GasPrice:      "",
			GasAdjustment: 1.5,
			TxTimeout:     60,
		},

		DANode: NodeConfig{
			ChainID:       "testnet-l1-1",
			Bech32Prefix:  "init",
			RPCAddress:    "tcp://localhost:26657",
			GasPrice:      "0.15uinit",
			GasAdjustment: 1.5,
			TxTimeout:     60,
		},

		BridgeExecutor:         "",
		DisableOutputSubmitter: false,
		DisableBatchSubmitter:  false,

		MaxChunks:         5000,
		MaxChunkSize:      300000,  // 300KB
		MaxSubmissionTime: 60 * 60, // 1 hour

		DisableAutoSetL1Height: false,
		L1StartHeight:          0,
		L2StartHeight:          0,
		BatchStartHeight:       0,
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

	if err := cfg.DANode.Validate(); err != nil {
		return err
	}

	if cfg.MaxChunks <= 0 {
		return errors.New("max chunks must be greater than 0")
	}

	if cfg.MaxChunkSize <= 0 {
		return errors.New("max chunk size must be greater than 0")
	}

	if cfg.MaxSubmissionTime <= 0 {
		return errors.New("max submission time must be greater than 0")
	}

	if cfg.L1StartHeight < 0 {
		return errors.New("l1 start height must be greater than or equal to 0")
	}

	if cfg.L2StartHeight < 0 {
		return errors.New("l2 start height must be greater than or equal to 0")
	}

	if cfg.BatchStartHeight < 0 {
		return errors.New("batch start height must be greater than or equal to 0")
	}
	return nil
}

func (cfg Config) L1NodeConfig(homePath string) nodetypes.NodeConfig {
	nc := nodetypes.NodeConfig{
		RPC:          cfg.L1Node.RPCAddress,
		ProcessType:  nodetypes.PROCESS_TYPE_DEFAULT,
		Bech32Prefix: cfg.L1Node.Bech32Prefix,
	}

	if !cfg.DisableOutputSubmitter {
		nc.BroadcasterConfig = &btypes.BroadcasterConfig{
			ChainID:       cfg.L1Node.ChainID,
			GasPrice:      cfg.L1Node.GasPrice,
			GasAdjustment: cfg.L1Node.GasAdjustment,
			TxTimeout:     time.Duration(cfg.L1Node.TxTimeout) * time.Second,
			Bech32Prefix:  cfg.L1Node.Bech32Prefix,
			HomePath:      homePath,
		}
	}

	return nc
}

func (cfg Config) L2NodeConfig(homePath string) nodetypes.NodeConfig {
	nc := nodetypes.NodeConfig{
		RPC:          cfg.L2Node.RPCAddress,
		ProcessType:  nodetypes.PROCESS_TYPE_DEFAULT,
		Bech32Prefix: cfg.L2Node.Bech32Prefix,
	}

	if cfg.BridgeExecutor != "" {
		nc.BroadcasterConfig = &btypes.BroadcasterConfig{
			ChainID:       cfg.L2Node.ChainID,
			GasPrice:      cfg.L2Node.GasPrice,
			GasAdjustment: cfg.L2Node.GasAdjustment,
			TxTimeout:     time.Duration(cfg.L2Node.TxTimeout) * time.Second,
			Bech32Prefix:  cfg.L2Node.Bech32Prefix,
			HomePath:      homePath,
		}
	}

	return nc
}

func (cfg Config) DANodeConfig(homePath string) nodetypes.NodeConfig {
	nc := nodetypes.NodeConfig{
		RPC:          cfg.DANode.RPCAddress,
		ProcessType:  nodetypes.PROCESS_TYPE_ONLY_BROADCAST,
		Bech32Prefix: cfg.DANode.Bech32Prefix,
	}

	if !cfg.DisableBatchSubmitter {
		nc.BroadcasterConfig = &btypes.BroadcasterConfig{
			ChainID:       cfg.DANode.ChainID,
			GasPrice:      cfg.DANode.GasPrice,
			GasAdjustment: cfg.DANode.GasAdjustment,
			TxTimeout:     time.Duration(cfg.DANode.TxTimeout) * time.Second,
			Bech32Prefix:  cfg.DANode.Bech32Prefix,
			HomePath:      homePath,
		}
	}
	return nc
}

func (cfg Config) BatchConfig() BatchConfig {
	return BatchConfig{
		MaxChunks:         cfg.MaxChunks,
		MaxChunkSize:      cfg.MaxChunkSize,
		MaxSubmissionTime: cfg.MaxSubmissionTime,
	}
}

type BatchConfig struct {
	MaxChunks         int64 `json:"max_chunks"`
	MaxChunkSize      int64 `json:"max_chunk_size"`
	MaxSubmissionTime int64 `json:"max_submission_time"` // seconds
}
