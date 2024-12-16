package e2e

import (
	"context"
	"testing"

	"cosmossdk.io/math"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestDeposit(t *testing.T) {
	l1ChainConfig := &ChainConfig{
		ChainID:        "initiation-2",
		Image:          ibc.DockerImage{Repository: "ghcr.io/initia-labs/initiad", Version: "v0.6.4", UIDGID: "1000:1000"},
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
		Image:          ibc.DockerImage{Repository: "ghcr.io/initia-labs/minimove", Version: "v0.6.5", UIDGID: "1000:1000"},
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
		ChainType:   ophosttypes.BatchInfo_CHAIN_TYPE_INITIA,
	}

	bridgeConfig := &BridgeConfig{
		SubmissionInterval:    "5s",
		FinalizationPeriod:    "10s",
		SubmissionStartHeight: "1",
		OracleEnabled:         true,
		Metadata:              "",
	}

	ctx := context.Background()

	op := SetupTest(t, ctx, BotExecutor, l1ChainConfig, l2ChainConfig, daChainConfig, bridgeConfig)

	l1User := interchaintest.GetAndFundTestUsers(t, ctx, "l1User", math.NewInt(100_000), op.Initia)[0]
	l2User := interchaintest.GetAndFundTestUsers(t, ctx, "l2User", math.NewInt(100_000), op.Minitia)[0]

	amount := sdk.NewCoin(op.Initia.Config().Denom, math.NewInt(1000))
	_, err := op.Initia.InitiateTokenDeposit(ctx, l1User.KeyName(), 1, l2User.FormattedAddress(), amount.String(), "")
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 5, op.Initia, op.Minitia)
	require.NoError(t, err)

	res, err := op.Initia.QueryTokenPairByL1Denom(ctx, 1, op.Initia.Config().Denom)
	require.NoError(t, err)

	l2UserBalance, err := op.Minitia.BankQueryBalance(ctx, l2User.FormattedAddress(), res.TokenPair.L2Denom)
	require.NoError(t, err)

	require.Equal(t, amount.Amount, l2UserBalance)

	prices, err := op.Minitia.QueryPrices(ctx, []string{"BTC/USD"})
	require.NoError(t, err)

	require.NotEqual(t, uint64(0), prices.Prices[0].Price.BlockHeight)
}
