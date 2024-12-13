package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"

	"cosmossdk.io/math"
	"github.com/icza/dyno"
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

	"github.com/cosmos/cosmos-sdk/crypto/keyring"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophostcli "github.com/initia-labs/OPinit/x/ophost/client/cli"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
)

const (
	BridgeExecutorKeyName  = "executor"
	L2ValidatorKeyName     = "validator"
	OutputSubmitterKeyName = "output"
	BatchSubmitterKeyName  = "batch"
	ChallengerKeyName      = "challenger"

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
	return &cfg
}

func MinitiaEncoding() *cosmostestutil.TestEncodingConfig {
	cfg := cosmos.DefaultEncoding()
	opchildtypes.RegisterInterfaces(cfg.InterfaceRegistry)
	return &cfg
}

func SetupTest(
	t *testing.T,
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

	ctx := context.Background()
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

	/// OPinit bot setup

	op := NewOPBot(logger, botName, t.Name(), client, network)
	t.Cleanup(func() {
		if err := op.Stop(ctx); err != nil {
			t.Logf("an error occurred while stopping the OP bot: %s", err)
		}
	})

	bridgeExecutor, err := op.AddKey(ctx, l2ChainConfig.ChainID, BridgeExecutorKeyName, l2ChainConfig.Bech32Prefix)
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
				Bin:            l1ChainConfig.Bin,
				Bech32Prefix:   l1ChainConfig.Bech32Prefix,
				Denom:          l1ChainConfig.Denom,
				Gas:            l1ChainConfig.Gas,
				GasPrices:      l1ChainConfig.GasPrices,
				GasAdjustment:  l1ChainConfig.GasAdjustment,
				TrustingPeriod: l1ChainConfig.TrustingPeriod,
				EncodingConfig: InitiaEncoding(),
				NoHostMount:    false,
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

	t.Cleanup(func() {
		_ = ic.Close()

		if relayer != nil {
			if err := relayer.StopRelayer(ctx, eRep); err != nil {
				t.Logf("an error occurred while stopping the relayer: %s", err)
			}
		}
	})

	require.NoError(t, ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{
		TestName:          t.Name(),
		Client:            client,
		NetworkID:         network,
		BlockDatabaseFile: interchaintest.DefaultBlockDatabaseFilepath(),

		SkipPathCreation: false,
	}))

	logger.Info("chains and relayer setup complete",
		zap.String("bridge executor", bridgeExecutor.FormattedAddress()),
		zap.String("l2 validator", l2Validator.FormattedAddress()),
		zap.String("output submitter", outputSubmitter.FormattedAddress()),
		zap.String("batch submitter", batchSubmitter.FormattedAddress()),
		zap.String("challenger", challenger.FormattedAddress()),
	)

	helper := OPTestHelper{
		logger,

		NewL1Chain(initia, outputSubmitter, batchSubmitter, challenger),
		NewL2Chain(minitia, bridgeExecutor, l2Validator),
		da,
		daChainConfig.ChainType,

		op,
		relayer,

		eRep,

		bridgeConfig,
	}

	// create bridge
	helper.CreateBridge(t, ctx)

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

func (op *OPTestHelper) CreateBridge(t *testing.T, ctx context.Context) {
	configBz, err := json.Marshal(op.BridgeConfig())
	require.NoError(t, err)

	// create bridge for initia

	// write bridge config to file
	fw := dockerutil.NewFileWriter(op.Logger, op.Initia.GetFullNode().DockerClient, t.Name())
	err = fw.WriteFile(ctx, op.Initia.GetFullNode().VolumeName, bridgeConfigPath, configBz)
	require.NoError(t, err)

	user0, err := op.Initia.BuildWallet(ctx, "user0", "")
	require.NoError(t, err)

	err = op.Initia.SendFunds(ctx, interchaintest.FaucetAccountKeyName, ibc.WalletAmount{
		Address: user0.FormattedAddress(),
		Amount:  math.NewInt(100_000),
		Denom:   op.Initia.Config().Denom,
	})
	require.NoError(t, err)

	_, err = op.Initia.CreateBridge(ctx, user0.KeyName(), path.Join(op.Initia.HomeDir(), bridgeConfigPath))
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
