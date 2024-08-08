package types

import (
	"errors"

	btypes "github.com/initia-labs/opinit-bots-go/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
)

type Config struct {
	// Version is the version used to build output root.
	Version uint8 `json:"version"`

	// ListenAddress is the address to listen for incoming requests.
	ListenAddress string `json:"listen_address"`

	L1RPCAddress string `json:"l1_rpc_address"`
	L2RPCAddress string `json:"l2_rpc_address"`
	DARPCAddress string `json:"da_rpc_address"`

	L1GasPrice string `json:"l1_gas_price"`
	L2GasPrice string `json:"l2_gas_price"`
	DAGasPrice string `json:"da_gas_price"`

	L1ChainID string `json:"l1_chain_id"`
	L2ChainID string `json:"l2_chain_id"`
	DAChainID string `json:"da_chain_id"`

	L1Bech32Prefix string `json:"l1_bech32_prefix"`
	L2Bech32Prefix string `json:"l2_bech32_prefix"`
	DABech32Prefix string `json:"da_bech32_prefix"`

	// OutputSubmitter is the key name in the keyring for the output submitter,
	// which is used to relay the output transaction from l2 to l1.
	//
	// If you don't want to use the output submitter feature, you can leave it empty.
	OutputSubmitter string `json:"output_submitter"`

	// BridgeExecutor is the key name in the keyring for the bridge executor,
	// which is used to relay initiate token bridge transaction from l1 to l2.
	//
	// If you don't want to use the bridge executor feature, you can leave it empty.
	BridgeExecutor string `json:"bridge_executor"`

	// RelayOracle is the flag to enable the oracle relay feature.
	RelayOracle bool `json:"relay_oracle"`

	// MaxChunks is the maximum number of chunks in a batch.
	MaxChunks int64 `json:"max_chunks"`
	// MaxChunkSize is the maximum size of a chunk in a batch.
	MaxChunkSize int64 `json:"max_chunk_size"`
	// MaxSubmissionTime is the maximum time to submit a batch.
	MaxSubmissionTime int64 `json:"max_submission_time"` // seconds

	// L2StartHeight is the height to start the l2 node. If it is 0, it will start from the latest height.
	// If the latest height stored in the db is not 0, this config is ignored.
	// L2 starts from the last submitted output l2 block number + 1 before L2StartHeight.
	// L1 starts from the block number of the output tx + 1
	L2StartHeight int64 `json:"l2_start_height"`
	// StartBatchHeight is the height to start the batch. If it is 0, it will start from the latest height.
	// If the latest height stored in the db is not 0, this config is ignored.
	BatchStartHeight int64 `json:"batch_start_height"`
}

type HostConfig struct {
	nodetypes.NodeConfig
	RelayOracle bool `json:"relay_oracle"`
}

func DefaultConfig() *Config {
	return &Config{
		Version:       1,
		ListenAddress: "localhost:3000",

		L1RPCAddress: "tcp://localhost:26657",
		L2RPCAddress: "tcp://localhost:27657",
		DARPCAddress: "tcp://localhost:28657",

		L1GasPrice: "0.15uinit",
		L2GasPrice: "",
		DAGasPrice: "",

		L1ChainID: "testnet-l1-1",
		L2ChainID: "testnet-l2-1",
		DAChainID: "testnet-da-1",

		L1Bech32Prefix: "init",
		L2Bech32Prefix: "init",
		DABech32Prefix: "init",

		OutputSubmitter: "",
		BridgeExecutor:  "",

		RelayOracle: true,

		MaxChunks:         5000,
		MaxChunkSize:      300000,  // 300KB
		MaxSubmissionTime: 60 * 60, // 1 hour

		L2StartHeight:    0,
		BatchStartHeight: 0,
	}
}

func (cfg Config) Validate() error {
	if cfg.Version == 0 {
		return errors.New("version is required")
	}

	if cfg.L1RPCAddress == "" {
		return errors.New("L1 RPC URL is required")
	}
	if cfg.L2RPCAddress == "" {
		return errors.New("L2 RPC URL is required")
	}
	if cfg.DARPCAddress == "" {
		return errors.New("L2 RPC URL is required")
	}

	if cfg.L1ChainID == "" {
		return errors.New("L1 chain ID is required")
	}
	if cfg.L2ChainID == "" {
		return errors.New("L2 chain ID is required")
	}
	if cfg.DAChainID == "" {
		return errors.New("DA chain ID is required")
	}

	if cfg.ListenAddress == "" {
		return errors.New("listen address is required")
	}

	if cfg.L1Bech32Prefix == "" {
		return errors.New("L1 bech32 prefix is required")
	}
	if cfg.L2Bech32Prefix == "" {
		return errors.New("L2 bech32 prefix is required")
	}
	if cfg.DABech32Prefix == "" {
		return errors.New("DA bech32 prefix is required")
	}

	return nil
}

func (cfg Config) L1NodeConfig(homePath string) nodetypes.NodeConfig {
	nc := nodetypes.NodeConfig{
		RPC:         cfg.L1RPCAddress,
		ProcessType: nodetypes.PROCESS_TYPE_DEFAULT,
	}

	if cfg.OutputSubmitter != "" {
		nc.BroadcasterConfig = &btypes.BroadcasterConfig{
			ChainID:      cfg.L1ChainID,
			GasPrice:     cfg.L1GasPrice,
			Bech32Prefix: cfg.L1Bech32Prefix,
			KeyringConfig: btypes.KeyringConfig{
				Name:     cfg.OutputSubmitter,
				HomePath: homePath,
			},
		}
	}

	return nc
}

func (cfg Config) L2NodeConfig(homePath string) nodetypes.NodeConfig {
	nc := nodetypes.NodeConfig{
		RPC:         cfg.L2RPCAddress,
		ProcessType: nodetypes.PROCESS_TYPE_DEFAULT,
	}

	if cfg.BridgeExecutor != "" {
		nc.BroadcasterConfig = &btypes.BroadcasterConfig{
			ChainID:      cfg.L2ChainID,
			GasPrice:     cfg.L2GasPrice,
			Bech32Prefix: cfg.L2Bech32Prefix,
			KeyringConfig: btypes.KeyringConfig{
				Name:     cfg.BridgeExecutor,
				HomePath: homePath,
			},
		}
	}

	return nc
}

func (cfg Config) DANodeConfig(homePath string) nodetypes.NodeConfig {
	nc := nodetypes.NodeConfig{
		RPC:         cfg.DARPCAddress,
		ProcessType: nodetypes.PROCESS_TYPE_DEFAULT,
		BroadcasterConfig: &btypes.BroadcasterConfig{
			ChainID:      cfg.DAChainID,
			GasPrice:     cfg.DAGasPrice,
			Bech32Prefix: cfg.DABech32Prefix,
			KeyringConfig: btypes.KeyringConfig{
				HomePath: homePath,
			},
		},
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
