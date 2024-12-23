package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/math"
	"cosmossdk.io/x/feegrant"
	"github.com/avast/retry-go/v4"
	"github.com/icza/dyno"
	oracleconfig "github.com/skip-mev/connect/v2/oracle/config"
	"github.com/skip-mev/connect/v2/providers/apis/marketmap"
	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/dockerutil"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testreporter"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"moul.io/zapfilter"

	cosmostestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/authz"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophostcli "github.com/initia-labs/OPinit/x/ophost/client/cli"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	bottypes "github.com/initia-labs/opinit-bots/bot/types"
	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	servertypes "github.com/initia-labs/opinit-bots/server/types"
)

var (
	RtyAttNum = uint(5)
	RtyAtt    = retry.Attempts(RtyAttNum)
	RtyDel    = retry.Delay(time.Millisecond * 100)
	RtyErr    = retry.LastErrorOnly(true)
)

const (
	BotExecutor   = "executor"
	BotChallenger = "challenger"

	BridgeExecutorKeyName       = "executor"
	OracleBridgeExecutorKeyName = "oracle"
	L2ValidatorKeyName          = "validator"
	OutputSubmitterKeyName      = "output"
	BatchSubmitterKeyName       = "batch"
	ChallengerKeyName           = "challenger"

	ibcPath = "initia-minitia"

	bridgeConfigPath = "bridge-config.json"
)

type ChainConfig struct {
	ChainID        string
	Image          ibc.DockerImage
	Bin            string
	Bech32Prefix   string
	Denom          string
	Gas            string
	GasPrices      string
	GasAdjustment  float64
	TrustingPeriod string
	NumValidators  int
	NumFullNodes   int
}

type DAChainConfig struct {
	ChainConfig
	ChainType ophosttypes.BatchInfo_ChainType
}

type BridgeConfig struct {
	SubmissionInterval    string
	FinalizationPeriod    string
	SubmissionStartHeight string
	OracleEnabled         bool
	Metadata              string
}

type OPTestHelper struct {
	Logger *zap.Logger

	Initia      *L1Chain
	Minitia     *L2Chain
	DA          *cosmos.CosmosChain
	DAChainType ophosttypes.BatchInfo_ChainType

	OP      *OPBot
	Relayer ibc.Relayer

	eRep *testreporter.RelayerExecReporter

	bridgeConfig *BridgeConfig
}

func InitiaEncoding() *cosmostestutil.TestEncodingConfig {
	cfg := cosmos.DefaultEncoding()
	ophosttypes.RegisterInterfaces(cfg.InterfaceRegistry)
	ophosttypes.RegisterLegacyAminoCodec(cfg.Amino)
	return &cfg
}

func MinitiaEncoding() *cosmostestutil.TestEncodingConfig {
	cfg := cosmos.DefaultEncoding()
	authz.RegisterInterfaces(cfg.InterfaceRegistry)
	feegrant.RegisterInterfaces(cfg.InterfaceRegistry)
	opchildtypes.RegisterInterfaces(cfg.InterfaceRegistry)
	return &cfg
}

func SetupTest(
	t *testing.T,
	ctx context.Context,
	botName string,
	l1ChainConfig *ChainConfig,
	l2ChainConfig *ChainConfig,
	daChainConfig *DAChainConfig,
	bridgeConfig *BridgeConfig,
) OPTestHelper {
	require.NotNil(t, l1ChainConfig)
	require.NotNil(t, l2ChainConfig)
	require.NotNil(t, daChainConfig)
	require.NotNil(t, bridgeConfig)

	client, network := interchaintest.DockerSetup(t)
	rep := testreporter.NewReporter(os.Stdout)
	eRep := rep.RelayerExecReporter(t)

	// logger setup

	logger := zaptest.NewLogger(t)
	filteringCore := zapfilter.NewFilteringCore(logger.Core(), func(entry zapcore.Entry, fields []zapcore.Field) bool {
		if entry.Level == zap.InfoLevel && entry.Message == "Failed to decode tx" {
			for _, field := range fields {
				if field.Key == "error" {
					if err := field.Interface.(error); strings.Contains(err.Error(), "expected 2 wire type, got 0") {
						return false
					}
				}
			}
		}
		return true
	})
	logger = zaptest.NewLogger(t, zaptest.WrapOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core { return filteringCore })))

	// OPinit bot setup

	op := NewOPBot(logger, botName, t.Name(), client, network)

	bridgeExecutor, err := op.AddKey(ctx, l2ChainConfig.ChainID, BridgeExecutorKeyName, l2ChainConfig.Bech32Prefix)
	require.NoError(t, err)
	oracleBridgeExecutor, err := op.AddKey(ctx, l2ChainConfig.ChainID, OracleBridgeExecutorKeyName, l2ChainConfig.Bech32Prefix)
	require.NoError(t, err)
	l2Validator, err := op.AddKey(ctx, l2ChainConfig.ChainID, L2ValidatorKeyName, l2ChainConfig.Bech32Prefix)
	require.NoError(t, err)

	outputSubmitter, err := op.AddKey(ctx, l1ChainConfig.ChainID, OutputSubmitterKeyName, l1ChainConfig.Bech32Prefix)
	require.NoError(t, err)
	batchSubmitter, err := op.AddKey(ctx, daChainConfig.ChainID, BatchSubmitterKeyName, daChainConfig.Bech32Prefix)
	require.NoError(t, err)
	challenger, err := op.AddKey(ctx, l1ChainConfig.ChainID, ChallengerKeyName, l1ChainConfig.Bech32Prefix)
	require.NoError(t, err)

	// chains setup

	specs := []*interchaintest.ChainSpec{
		{
			ChainConfig: ibc.ChainConfig{
				Type:    "cosmos",
				Name:    "initia",
				ChainID: l1ChainConfig.ChainID,
				Images: []ibc.DockerImage{
					l1ChainConfig.Image,
				},
				SidecarConfigs: []ibc.SidecarConfig{
					{
						ProcessName: "connect",
						HomeDir:     "/oracle",
						Image: ibc.DockerImage{
							Repository: "ghcr.io/skip-mev/connect-sidecar",
							Version:    "v2.0.1",
							UIDGID:     "1000:1000",
						},
						Ports: []string{
							"8080",
						},
						StartCmd: []string{
							"connect",
							"--oracle-config", "/oracle/oracle.json",
						},
					},
				},
				Bin:            l1ChainConfig.Bin,
				Bech32Prefix:   l1ChainConfig.Bech32Prefix,
				Denom:          l1ChainConfig.Denom,
				Gas:            l1ChainConfig.Gas,
				GasPrices:      l1ChainConfig.GasPrices,
				GasAdjustment:  l1ChainConfig.GasAdjustment,
				TrustingPeriod: l1ChainConfig.TrustingPeriod,
				EncodingConfig: InitiaEncoding(),
				NoHostMount:    false,
				PreGenesis: func(ch ibc.Chain) error {
					l1Chain := ch.(*cosmos.CosmosChain)

					cfg := marketmap.DefaultAPIConfig
					cfg.Endpoints = []oracleconfig.Endpoint{
						{
							URL: fmt.Sprintf("%s:9090", l1Chain.Validators[0].HostName()),
						},
					}

					// Create the oracle config
					oracleConfig := oracleconfig.OracleConfig{
						UpdateInterval: 500 * time.Millisecond,
						MaxPriceAge:    1 * time.Minute,
						Host:           "0.0.0.0",
						Port:           "8080",
						Providers: map[string]oracleconfig.ProviderConfig{
							marketmap.Name: {
								Name: marketmap.Name,
								API:  cfg,
								Type: "market_map_provider",
							},
						},
					}

					oracleConfigBz, err := json.Marshal(oracleConfig)
					require.NoError(t, err)

					err = l1Chain.Sidecars[0].WriteFile(ctx, oracleConfigBz, "oracle.json")
					require.NoError(t, err)

					ctx := context.Background()
					c := make(testutil.Toml)
					oracle := make(testutil.Toml)
					oracle["enabled"] = "true"
					oracle["oracle_address"] = fmt.Sprintf("%s:8080", l1Chain.Sidecars[0].HostName())
					c["oracle"] = oracle

					err = testutil.ModifyTomlConfigFile(ctx, logger, client, t.Name(), l1Chain.Validators[0].VolumeName, "config/app.toml", c)
					require.NoError(t, err)
					return nil
				},
			},
			NumValidators: &l1ChainConfig.NumValidators,
			NumFullNodes:  &l1ChainConfig.NumFullNodes,
		},
		{
			ChainConfig: ibc.ChainConfig{
				Type:    "cosmos",
				Name:    "minitia",
				ChainID: l2ChainConfig.ChainID,
				Images: []ibc.DockerImage{
					l2ChainConfig.Image,
				},
				Bin:            l2ChainConfig.Bin,
				Bech32Prefix:   l2ChainConfig.Bech32Prefix,
				Denom:          l2ChainConfig.Denom,
				Gas:            l2ChainConfig.Gas,
				GasPrices:      l2ChainConfig.GasPrices,
				GasAdjustment:  l2ChainConfig.GasAdjustment,
				TrustingPeriod: l2ChainConfig.TrustingPeriod,
				EncodingConfig: MinitiaEncoding(),
				NoHostMount:    false,
				SkipGenTx:      true,
				PreGenesis: func(ch ibc.Chain) error {
					ctx := context.Background()
					c := make(testutil.Toml)
					consensus := make(testutil.Toml)
					consensus["create_empty_blocks"] = true
					consensus["create_empty_blocks_interval"] = "500ms"
					c["consensus"] = consensus

					l2Chain := ch.(*cosmos.CosmosChain)

					err := testutil.ModifyTomlConfigFile(ctx, logger, client, t.Name(), l2Chain.Validators[0].VolumeName, "config/config.toml", c)
					require.NoError(t, err)

					_, err = ch.BuildWallet(ctx, bridgeExecutor.KeyName(), bridgeExecutor.Mnemonic())
					require.NoError(t, err)

					_, err = ch.BuildWallet(ctx, l2Validator.KeyName(), l2Validator.Mnemonic())
					require.NoError(t, err)

					_, _, err = ch.Exec(ctx, []string{"minitiad", "genesis", "add-genesis-account", bridgeExecutor.KeyName(), "", "--home", ch.HomeDir(), "--keyring-backend", keyring.BackendTest}, nil)
					require.NoError(t, err)

					_, _, err = ch.Exec(ctx, []string{"minitiad", "genesis", "add-genesis-validator", l2Validator.KeyName(), "--home", ch.HomeDir(), "--keyring-backend", keyring.BackendTest}, nil)
					require.NoError(t, err)
					return nil
				},
				ModifyGenesis: func(cfg ibc.ChainConfig, genbz []byte) ([]byte, error) {
					g := make(map[string]interface{})
					if err := json.Unmarshal(genbz, &g); err != nil {
						return nil, fmt.Errorf("failed to unmarshal genesis file: %w", err)
					}

					if err := dyno.Set(g, []string{
						bridgeExecutor.FormattedAddress(),
					},
						"app_state", "opchild", "params", "bridge_executors",
					); err != nil {
						return nil, err
					}

					if err := dyno.Set(g, []string{
						bridgeExecutor.FormattedAddress(),
						l2Validator.FormattedAddress(),
					},
						"app_state", "opchild", "params", "fee_whitelist",
					); err != nil {
						return nil, err
					}

					out, err := json.Marshal(g)
					if err != nil {
						return nil, fmt.Errorf("failed to marshal genesis bytes to json: %w", err)
					}
					return out, nil
				},
			},
			NumValidators: &l2ChainConfig.NumValidators,
			NumFullNodes:  &l2ChainConfig.NumFullNodes,
		},
	}

	if daChainConfig.ChainType == ophosttypes.BatchInfo_CHAIN_TYPE_CELESTIA {
		specs = append(specs, &interchaintest.ChainSpec{
			ChainConfig: ibc.ChainConfig{
				Type:    "cosmos",
				Name:    "da",
				ChainID: daChainConfig.ChainID,
				Images: []ibc.DockerImage{
					daChainConfig.Image,
				},
				Bin:            daChainConfig.Bin,
				Bech32Prefix:   daChainConfig.Bech32Prefix,
				Denom:          daChainConfig.Denom,
				Gas:            daChainConfig.Gas,
				GasPrices:      daChainConfig.GasPrices,
				GasAdjustment:  daChainConfig.GasAdjustment,
				TrustingPeriod: daChainConfig.TrustingPeriod,
				NoHostMount:    false,
			},
			NumValidators: &daChainConfig.NumValidators,
			NumFullNodes:  &daChainConfig.NumFullNodes,
		})
	}
	cf := interchaintest.NewBuiltinChainFactory(logger, specs)

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	initia, minitia := chains[0].(*cosmos.CosmosChain), chains[1].(*cosmos.CosmosChain)
	da := initia
	if len(chains) == 3 {
		da = chains[2].(*cosmos.CosmosChain)
	}

	// relayer setup

	relayer := interchaintest.NewBuiltinRelayerFactory(ibc.CosmosRly, zaptest.NewLogger(t)).Build(t, client, network)

	ic := interchaintest.NewInterchain().
		AddChain(initia).
		AddChain(minitia).
		AddRelayer(relayer, "relayer").
		AddLink(interchaintest.InterchainLink{
			Chain1:  initia,
			Chain2:  minitia,
			Relayer: relayer,
			Path:    ibcPath,
		})

	icBuildOptions := interchaintest.InterchainBuildOptions{
		TestName:          t.Name(),
		Client:            client,
		NetworkID:         network,
		BlockDatabaseFile: interchaintest.DefaultBlockDatabaseFilepath(),

		SkipPathCreation: false,
	}
	require.NoError(t, ic.Build(ctx, eRep, icBuildOptions))

	err = initia.StartAllSidecars(ctx)
	require.NoError(t, err)

	err = relayer.StartRelayer(ctx, eRep)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = ic.Close()

		if relayer != nil {
			if err := relayer.StopRelayer(ctx, eRep); err != nil {
				t.Logf("an error occurred while stopping the relayer: %s", err)
			}
		}
	})

	err = initia.SendFunds(ctx, interchaintest.FaucetAccountKeyName, ibc.WalletAmount{
		Address: outputSubmitter.FormattedAddress(),
		Denom:   initia.Config().Denom,
		Amount:  math.NewInt(100_000_000_000),
	})
	require.NoError(t, err)

	err = initia.SendFunds(ctx, interchaintest.FaucetAccountKeyName, ibc.WalletAmount{
		Address: batchSubmitter.FormattedAddress(),
		Denom:   initia.Config().Denom,
		Amount:  math.NewInt(100_000_000_000),
	})
	require.NoError(t, err)

	err = initia.SendFunds(ctx, interchaintest.FaucetAccountKeyName, ibc.WalletAmount{
		Address: challenger.FormattedAddress(),
		Denom:   initia.Config().Denom,
		Amount:  math.NewInt(100_000_000_000),
	})
	require.NoError(t, err)

	logger.Info("chains and relayer setup complete",
		zap.String("bridge executor", bridgeExecutor.FormattedAddress()),
		zap.String("oracle bridge executor", oracleBridgeExecutor.FormattedAddress()),
		zap.String("l2 validator", l2Validator.FormattedAddress()),
		zap.String("output submitter", outputSubmitter.FormattedAddress()),
		zap.String("batch submitter", batchSubmitter.FormattedAddress()),
		zap.String("challenger", challenger.FormattedAddress()),
	)

	helper := OPTestHelper{
		logger,

		NewL1Chain(logger, initia, outputSubmitter, batchSubmitter, challenger),
		NewL2Chain(logger, minitia, bridgeExecutor, oracleBridgeExecutor, l2Validator),
		da,
		daChainConfig.ChainType,

		op,
		relayer,

		eRep,

		bridgeConfig,
	}

	// create bridge
	helper.CreateBridge(t, ctx)
	helper.SetOPConfig(t, ctx)

	// register validators on l2 chain
	err = relayer.UpdateClients(ctx, eRep, ibcPath)
	require.NoError(t, err)

	err = op.Start(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := op.Stop(ctx); err != nil {
			t.Logf("an error occurred while stopping the OP bot: %s", err)
		}

		if op.customImage != nil {
			err = DestroyOPBotImage(op.customImage.Repository)
			if err != nil {
				t.Logf("an error occurred while stopping the OP bot: %s", err)
			}
		}
	})
	err = op.WaitForSync(ctx)
	require.NoError(t, err)

	return helper
}

func (op OPTestHelper) BridgeConfig() ophostcli.BridgeCliConfig {
	return ophostcli.BridgeCliConfig{
		Challenger: op.Initia.Challenger.FormattedAddress(),
		Proposer:   op.Initia.OutputSubmitter.FormattedAddress(),
		BatchInfo: ophosttypes.BatchInfo{
			Submitter: op.Initia.BatchSubmitter.FormattedAddress(),
			ChainType: op.DAChainType,
		},
		SubmissionInterval:    op.bridgeConfig.SubmissionInterval,
		FinalizationPeriod:    op.bridgeConfig.FinalizationPeriod,
		SubmissionStartHeight: op.bridgeConfig.SubmissionStartHeight,
		OracleEnabled:         op.bridgeConfig.OracleEnabled,
		Metadata:              op.bridgeConfig.Metadata,
	}
}

func (op OPTestHelper) SetOPConfig(t *testing.T, ctx context.Context) {
	var cfg bottypes.Config
	switch op.OP.botName {
	case BotExecutor:
		cfg = op.ExecutorConfig()
	case BotChallenger:
		t.Fatal("challenger bot not supported")
	default:
		t.Fatalf("unknown bot name: %s", op.OP.botName)
	}

	configBz, err := json.Marshal(cfg)
	require.NoError(t, err)

	configName := fmt.Sprintf("%s.json", op.OP.botName)

	err = op.OP.WriteFileToHomeDir(ctx, configName, configBz)
	require.NoError(t, err)

	if op.bridgeConfig.OracleEnabled && op.OP.botName == BotExecutor {
		// grant oracle permissions
		err := op.OP.GrantOraclePermissions(ctx, op.Minitia.OracleBridgeExecutor.FormattedAddress())
		require.NoError(t, err)
		err = testutil.WaitForBlocks(ctx, 2, op.Minitia.GetFullNode())
		require.NoError(t, err)
	}
}

func (op OPTestHelper) ExecutorConfig() *executortypes.Config {
	return &executortypes.Config{
		Version: 1,

		Server: servertypes.ServerConfig{
			Address:      "0.0.0.0:3000",
			AllowOrigins: "*",
			AllowHeaders: "Origin, Content-Type, Accept",
			AllowMethods: "GET",
		},

		L1Node: executortypes.NodeConfig{
			ChainID:       op.Initia.Config().ChainID,
			Bech32Prefix:  op.Initia.Config().Bech32Prefix,
			RPCAddress:    fmt.Sprintf("http://%s:26657", op.Initia.GetFullNode().Name()),
			GasPrice:      op.Initia.Config().GasPrices,
			GasAdjustment: op.Initia.Config().GasAdjustment,
			TxTimeout:     10,
		},

		L2Node: executortypes.NodeConfig{
			ChainID:       op.Minitia.Config().ChainID,
			Bech32Prefix:  op.Minitia.Config().Bech32Prefix,
			RPCAddress:    fmt.Sprintf("http://%s:26657", op.Minitia.GetFullNode().Name()),
			GasPrice:      "",
			GasAdjustment: op.Minitia.Config().GasAdjustment,
			TxTimeout:     10,
		},

		DANode: executortypes.NodeConfig{
			ChainID:       op.DA.Config().ChainID,
			Bech32Prefix:  op.DA.Config().Bech32Prefix,
			RPCAddress:    fmt.Sprintf("http://%s:26657", op.DA.GetFullNode().Name()),
			GasPrice:      op.DA.Config().GasPrices,
			GasAdjustment: op.DA.Config().GasAdjustment,
			TxTimeout:     10,
		},

		BridgeExecutor:         op.Minitia.BridgeExecutor.KeyName(),
		OracleBridgeExecutor:   op.Minitia.OracleBridgeExecutor.KeyName(),
		DisableOutputSubmitter: false,
		DisableBatchSubmitter:  false,

		MaxChunks:         5000,
		MaxChunkSize:      300000, // 300KB
		MaxSubmissionTime: 10,     // 10 seconds

		DisableAutoSetL1Height:        false,
		L1StartHeight:                 1,
		L2StartHeight:                 1,
		BatchStartHeight:              1,
		DisableDeleteFutureWithdrawal: false,
	}
}

func (op *OPTestHelper) CreateBridge(t *testing.T, ctx context.Context) {
	configBz, err := json.Marshal(op.BridgeConfig())
	require.NoError(t, err)

	// create bridge for initia

	// write bridge config to file
	fw := dockerutil.NewFileWriter(op.Logger, op.Initia.GetFullNode().DockerClient, t.Name())
	err = fw.WriteFile(ctx, op.Initia.GetFullNode().VolumeName, bridgeConfigPath, configBz)
	require.NoError(t, err)

	user := interchaintest.GetAndFundTestUsers(t, ctx, "user", math.NewInt(100_000), op.Initia)[0]
	_, err = op.Initia.CreateBridge(ctx, user.KeyName(), path.Join(op.Initia.HomeDir(), bridgeConfigPath))
	require.NoError(t, err)

	res, err := op.Initia.QueryBridge(ctx, 1)
	require.NoError(t, err)

	op.Logger.Info("bridge created", zap.Uint64("bridge_id", res.BridgeId), zap.String("bridge_addr", res.BridgeAddr), zap.String("chain_id", op.Initia.Config().ChainID))

	require.Equal(t, uint64(1), res.BridgeId)

	// set bridge info for minitia

	// write bridge config to file
	fw = dockerutil.NewFileWriter(op.Logger, op.Minitia.GetFullNode().DockerClient, t.Name())
	err = fw.WriteFile(ctx, op.Minitia.GetFullNode().VolumeName, bridgeConfigPath, configBz)
	require.NoError(t, err)

	clients, err := op.Relayer.GetClients(ctx, op.eRep, op.Minitia.Config().ChainID)
	require.NoError(t, err)

	_, err = op.Minitia.SetBridgeInfo(ctx, res.BridgeId, res.BridgeAddr, op.Initia.Config().ChainID, clients[0].ClientID, path.Join(op.Minitia.HomeDir(), bridgeConfigPath))
	require.NoError(t, err)
}
