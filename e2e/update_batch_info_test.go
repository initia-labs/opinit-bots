package e2e

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/math"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
)

func TestUpdateBatchInfo(t *testing.T) {
	l1ChainConfig := &ChainConfig{
		ChainID:        "initiation-2",
		Image:          ibc.DockerImage{Repository: "ghcr.io/initia-labs/initiad", Version: "v0.7.2", UIDGID: "1000:1000"},
		Bin:            "initiad",
		Bech32Prefix:   "init",
		Denom:          "uinit",
		Gas:            "auto",
		GasPrices:      "0.025uinit",
		GasAdjustment:  1.2,
		TrustingPeriod: "1h",
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
		TrustingPeriod: "1h",
		NumValidators:  1,
		NumFullNodes:   0,
	}

	bridgeConfig := &BridgeConfig{
		SubmissionInterval:    "5s",
		FinalizationPeriod:    "10s",
		SubmissionStartHeight: "1",
		OracleEnabled:         true,
		Metadata:              "",
	}

	daChainConfig := &DAChainConfig{
		ChainConfig: ChainConfig{
			ChainID:        "celestia",
			Image:          ibc.DockerImage{Repository: "ghcr.io/celestiaorg/celestia-app", Version: "v3.3.1", UIDGID: "10001:10001"},
			Bin:            "celestia-appd",
			Bech32Prefix:   "celestia",
			Denom:          "utia",
			Gas:            "auto",
			GasPrices:      "0.25utia",
			GasAdjustment:  1.5,
			TrustingPeriod: "1h",
			NumValidators:  1,
			NumFullNodes:   0,
		},
		ChainType: ophosttypes.BatchInfo_CELESTIA,
	}

	ctx := context.Background()

	op := SetupTest(t, ctx, BotExecutor, l1ChainConfig, l2ChainConfig, daChainConfig, bridgeConfig, ibc.CosmosRly)

	err := testutil.WaitForBlocks(ctx, 20, op.Initia, op.Minitia)
	require.NoError(t, err)

	newDaChainConfig := &DAChainConfig{
		ChainConfig: *l1ChainConfig,
		ChainType:   ophosttypes.BatchInfo_INITIA,
	}
	newBatchSubmitter, err := op.OP.AddKey(ctx, newDaChainConfig.ChainID, BatchSubmitterKeyName, newDaChainConfig.Bech32Prefix)
	require.NoError(t, err)
	newDAChain := NewDAChain(op.Logger, op.Initia.CosmosChain, newDaChainConfig.ChainType, newBatchSubmitter)

	err = newDAChain.SendFunds(ctx, interchaintest.FaucetAccountKeyName, ibc.WalletAmount{
		Address: newDAChain.BatchSubmitter.FormattedAddress(),
		Denom:   newDAChain.Config().Denom,
		Amount:  math.NewInt(100_000_000_000),
	})
	require.NoError(t, err)

	oldDAChain := op.ChangeDA(t, ctx, newDAChain)
	err = op.OP.Pause(ctx)
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 3, op.Initia, op.Minitia)
	require.NoError(t, err)

	_, err = op.OP.UpdateBatchInfo(ctx, newDaChainConfig.ChainType.String(), newDAChain.BatchSubmitter.FormattedAddress())
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 2, op.Initia.GetFullNode())
	require.NoError(t, err)
	batchInfos, err := op.Initia.QueryBatchInfos(ctx, 1)
	require.NoError(t, err)
	require.Len(t, batchInfos, 2)
	require.Equal(t, batchInfos[1].BatchInfo.ChainType, newDaChainConfig.ChainType)
	require.Equal(t, batchInfos[1].BatchInfo.Submitter, newDAChain.BatchSubmitter.FormattedAddress())

	err = op.OP.Resume(ctx)
	require.NoError(t, err)

	// OP Bot will stop with panic message
	err = op.OP.WaitForStop(ctx, 30*time.Second)
	require.NoError(t, err)
	err = op.OP.Stop(ctx)
	require.NoError(t, err)

	bridgeInfo, err := op.Minitia.QueryBridgeInfo(ctx)
	require.NoError(t, err)
	require.Equal(t, bridgeInfo.BridgeInfo.BridgeConfig.BatchInfo.Submitter, newDAChain.BatchSubmitter.FormattedAddress())
	require.Equal(t, bridgeInfo.BridgeInfo.BridgeConfig.BatchInfo.ChainType.String(), newDaChainConfig.ChainType.String())

	oldDABatchData, err := oldDAChain.QueryBatchData(ctx)
	require.NoError(t, err)

	err = op.OP.Start(ctx)
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 20, op.Initia, op.Minitia)
	require.NoError(t, err)

	newBatches, err := op.DA.QueryBatchData(ctx)
	require.NoError(t, err)
	require.Greater(t, len(newBatches), 0)

	op.ChangeDA(t, ctx, oldDAChain)
	err = op.OP.Pause(ctx)
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 3, op.Initia, op.Minitia)
	require.NoError(t, err)

	_, err = op.OP.UpdateBatchInfo(ctx, oldDAChain.ChainType.String(), oldDAChain.BatchSubmitter.FormattedAddress())
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 2, op.Initia.GetFullNode())
	require.NoError(t, err)
	err = op.OP.Resume(ctx)
	require.NoError(t, err)

	// OP Bot will stop with panic message
	err = op.OP.WaitForStop(ctx, 30*time.Second)
	require.NoError(t, err)
	err = op.OP.Stop(ctx)
	require.NoError(t, err)

	err = op.OP.Start(ctx)
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 20, op.Initia, op.Minitia)
	require.NoError(t, err)

	newBatches, err = op.DA.QueryBatchData(ctx)
	require.NoError(t, err)
	require.Greater(t, len(newBatches), len(oldDABatchData))

	bridgeInfo, err = op.Minitia.QueryBridgeInfo(ctx)
	require.NoError(t, err)

	require.Equal(t, bridgeInfo.BridgeInfo.BridgeConfig.BatchInfo.Submitter, oldDAChain.BatchSubmitter.FormattedAddress())
	require.Equal(t, bridgeInfo.BridgeInfo.BridgeConfig.BatchInfo.ChainType.String(), oldDAChain.ChainType.String())
}
