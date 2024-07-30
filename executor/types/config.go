package types

import (
	"errors"

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

	// OutputSubmitterMnemonic is the mnemonic phrase for the output submitter,
	// which is used to relay the output transaction from l2 to l1.
	//
	// If you don't want to use the output submitter feature, you can leave it empty.
	OutputSubmitterMnemonic string `json:"output_submitter_mnemonic"`

	// BridgeExecutorMnemonic is the mnemonic phrase for the bridge executor,
	// which is used to relay initiate token bridge transaction from l1 to l2.
	//
	// If you don't want to use the bridge executor feature, you can leave it empty.
	BridgeExecutorMnemonic string `json:"bridge_executor_mnemonic"`

	// BatchSubmitterMnemonic is the mnemonic phrase for the batch submitter,
	// which is used to relay the batch of blocks from l2 to da.
	//
	// If you don't want to use the batch submitter feature, you can leave it empty.
	BatchSubmitterMnemonic string `json:"batch_submitter_mnemonic"`

	// RelayOracle is the flag to enable the oracle relay feature.
	RelayOracle bool `json:"relay_oracle"`

	MaxChunks         int64 `json:"max_chunks"`
	MaxChunkSize      int64 `json:"max_chunk_size"`
	MaxSubmissionTime int64 `json:"max_submission_time"` // seconds
}

type HostConfig struct {
	nodetypes.NodeConfig
	RelayOracle bool `json:"relay_oracle"`
}

func DefaultConfig() *Config {
	return &Config{
		Version:       1,
		ListenAddress: "tcp://localhost:3000",

		L1RPCAddress: "tcp://localhost:26657",
		L2RPCAddress: "tcp://localhost:27657",
		DARPCAddress: "tcp://localhost:28657",

		L1GasPrice: "0.15uinit",
		L2GasPrice: "",
		DAGasPrice: "",

		L1ChainID: "testnet-l1-1",
		L2ChainID: "testnet-l2-1",
		DAChainID: "testnet-l3-1",

		OutputSubmitterMnemonic: "",
		BridgeExecutorMnemonic:  "",
		BatchSubmitterMnemonic:  "",

		RelayOracle: true,

		MaxChunks:         5000,
		MaxChunkSize:      300000,  // 300KB
		MaxSubmissionTime: 60 * 60, // 1 hour
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
		return errors.New("L2 RPC URL is required")
	}
	if cfg.ListenAddress == "" {
		return errors.New("listen address is required")
	}
	return nil
}

func (cfg Config) L1NodeConfig() nodetypes.NodeConfig {
	return nodetypes.NodeConfig{
		RPC:      cfg.L1RPCAddress,
		ChainID:  cfg.L1ChainID,
		Mnemonic: cfg.OutputSubmitterMnemonic,
	}
}

func (cfg Config) L2NodeConfig() nodetypes.NodeConfig {
	return nodetypes.NodeConfig{
		RPC:      cfg.L2RPCAddress,
		ChainID:  cfg.L2ChainID,
		Mnemonic: cfg.BridgeExecutorMnemonic,
	}
}

func (cfg Config) DANodeConfig() nodetypes.NodeConfig {
	return nodetypes.NodeConfig{
		RPC:      cfg.DARPCAddress,
		ChainID:  cfg.DAChainID,
		Mnemonic: cfg.BatchSubmitterMnemonic,
	}
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
