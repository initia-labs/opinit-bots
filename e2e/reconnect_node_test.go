package e2e

import (
	"context"
	"testing"
	"time"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/stretchr/testify/require"
)

func TestReconnectNodes(t *testing.T) {
	l1ChainConfig := &ChainConfig{
		ChainID:        "initiation-2",
		Image:          ibc.DockerImage{Repository: "ghcr.io/initia-labs/initiad", Version: "v0.7.2", UIDGID: "1000:1000"},
		Bin:            "initiad",
		Bech32Prefix:   "init",
		Denom:          "uinit",
		Gas:            "auto",
		GasPrices:      "0.025uinit",
		GasAdjustment:  1.2,
		TrustingPeriod: "168h",
		NumValidators:  1,
		NumFullNodes:   0,
	}

	l2ChainConfig := &ChainConfig{
		ChainID:        "minimove-2",
		Image:          ibc.DockerImage{Repository: "ghcr.io/initia-labs/minimove", Version: "v0.7.0-1", UIDGID: "1000:1000"},
		Bin:            "minitiad",
		Bech32Prefix:   "init",
		Denom:          "umin",
		Gas:            "auto",
		GasPrices:      "0.025umin",
		GasAdjustment:  1.2,
		TrustingPeriod: "168h",
		NumValidators:  1,
		NumFullNodes:   0,
	}

	daChainConfig := &DAChainConfig{
		ChainConfig: *l1ChainConfig,
		ChainType:   ophosttypes.BatchInfo_INITIA,
	}

	bridgeConfig := &BridgeConfig{
		SubmissionInterval:    "5s",
		FinalizationPeriod:    "10s",
		SubmissionStartHeight: "1",
		OracleEnabled:         true,
		Metadata:              "",
	}

	ctx := context.Background()

	op := SetupTest(t, ctx, BotExecutor, l1ChainConfig, l2ChainConfig, daChainConfig, bridgeConfig, ibc.CosmosRly)
	err := op.Relayer.PauseRelayer(ctx)
	require.NoError(t, err)

	// pause only l1 chain
	err = op.Initia.GetFullNode().PauseContainer(ctx)
	require.NoError(t, err)
	status0, err := op.OP.QueryExecutorStatus()
	require.NoError(t, err)

	time.Sleep(10 * time.Second)

	// resume l1 chain
	err = op.Initia.GetFullNode().UnpauseContainer(ctx)
	require.NoError(t, err)

	time.Sleep(20 * time.Second)

	status1, err := op.OP.QueryExecutorStatus()
	require.NoError(t, err)

	require.Greater(t, *status1.Host.Node.LastBlockHeight, *status0.Host.Node.LastBlockHeight)

	prices, err := op.Minitia.QueryPrices(ctx, []string{"BTC/USD"})
	require.NoError(t, err)
	require.Greater(t, int64(prices.Prices[0].Price.BlockHeight), *status0.Host.Node.LastBlockHeight)

	// pause only l2 chain
	err = op.Minitia.GetFullNode().PauseContainer(ctx)
	require.NoError(t, err)
	status0, err = op.OP.QueryExecutorStatus()
	require.NoError(t, err)

	time.Sleep(10 * time.Second)

	// resume l2 chain
	err = op.Minitia.GetFullNode().UnpauseContainer(ctx)
	require.NoError(t, err)

	time.Sleep(20 * time.Second)

	status1, err = op.OP.QueryExecutorStatus()
	require.NoError(t, err)

	require.Greater(t, *status1.Child.Node.LastBlockHeight, *status0.Child.Node.LastBlockHeight)
	prices, err = op.Minitia.QueryPrices(ctx, []string{"BTC/USD"})
	require.NoError(t, err)
	require.Greater(t, int64(prices.Prices[0].Price.BlockHeight), *status0.Host.Node.LastBlockHeight)

	// pause both chains
	err = op.Initia.GetFullNode().PauseContainer(ctx)
	require.NoError(t, err)
	err = op.Minitia.GetFullNode().PauseContainer(ctx)
	require.NoError(t, err)
	status0, err = op.OP.QueryExecutorStatus()
	require.NoError(t, err)

	time.Sleep(10 * time.Second)

	// resume the chains
	err = op.Initia.GetFullNode().UnpauseContainer(ctx)
	require.NoError(t, err)
	err = op.Minitia.GetFullNode().UnpauseContainer(ctx)
	require.NoError(t, err)

	time.Sleep(20 * time.Second)

	status1, err = op.OP.QueryExecutorStatus()
	require.NoError(t, err)

	require.Greater(t, *status1.Host.Node.LastBlockHeight, *status0.Host.Node.LastBlockHeight)
	require.Greater(t, *status1.Child.Node.LastBlockHeight, *status0.Child.Node.LastBlockHeight)
	prices, err = op.Minitia.QueryPrices(ctx, []string{"BTC/USD"})
	require.NoError(t, err)
	require.Greater(t, int64(prices.Prices[0].Price.BlockHeight), *status0.Host.Node.LastBlockHeight)
}
