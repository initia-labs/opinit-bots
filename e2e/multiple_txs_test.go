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

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestMultipleDepositsAndWithdrawals(t *testing.T) {
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
		OracleEnabled:         false,
		Metadata:              "",
	}

	ctx := context.Background()

	op := SetupTest(t, ctx, BotExecutor, l1ChainConfig, l2ChainConfig, daChainConfig, bridgeConfig, ibc.CosmosRly)

	user0 := interchaintest.GetAndFundTestUsers(t, ctx, "user", math.NewInt(100_000), op.Initia, op.Minitia)
	user1 := interchaintest.GetAndFundTestUsers(t, ctx, "user", math.NewInt(100_000), op.Initia, op.Minitia)
	user2 := interchaintest.GetAndFundTestUsers(t, ctx, "user", math.NewInt(100_000), op.Initia, op.Minitia)
	user3 := interchaintest.GetAndFundTestUsers(t, ctx, "user", math.NewInt(100_000), op.Initia, op.Minitia)
	user4 := interchaintest.GetAndFundTestUsers(t, ctx, "user", math.NewInt(100_000), op.Initia, op.Minitia)
	amount := sdk.NewCoin(op.Initia.Config().Denom, math.NewInt(1000))

	_, err := op.Initia.InitiateTokenDeposit(ctx, user0[0].KeyName(), 1, user1[1].FormattedAddress(), amount.String(), "", true)
	require.NoError(t, err)
	_, err = op.Initia.InitiateTokenDeposit(ctx, user1[0].KeyName(), 1, user2[1].FormattedAddress(), amount.String(), "", true)
	require.NoError(t, err)
	_, err = op.Initia.InitiateTokenDeposit(ctx, user2[0].KeyName(), 1, user3[1].FormattedAddress(), amount.String(), "", true)
	require.NoError(t, err)
	_, err = op.Initia.InitiateTokenDeposit(ctx, user3[0].KeyName(), 1, user4[1].FormattedAddress(), amount.String(), "", true)
	require.NoError(t, err)
	_, err = op.Initia.InitiateTokenDeposit(ctx, user4[0].KeyName(), 1, user0[1].FormattedAddress(), amount.String(), "", true)
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 5, op.Initia, op.Minitia)
	require.NoError(t, err)

	res, err := op.Initia.QueryTokenPairByL1Denom(ctx, 1, op.Initia.Config().Denom)
	require.NoError(t, err)

	user0Balance, err := op.Minitia.BankQueryBalance(ctx, user0[1].FormattedAddress(), res.TokenPair.L2Denom)
	require.NoError(t, err)
	user1Balance, err := op.Minitia.BankQueryBalance(ctx, user1[1].FormattedAddress(), res.TokenPair.L2Denom)
	require.NoError(t, err)
	user2Balance, err := op.Minitia.BankQueryBalance(ctx, user2[1].FormattedAddress(), res.TokenPair.L2Denom)
	require.NoError(t, err)
	user3Balance, err := op.Minitia.BankQueryBalance(ctx, user3[1].FormattedAddress(), res.TokenPair.L2Denom)
	require.NoError(t, err)
	user4Balance, err := op.Minitia.BankQueryBalance(ctx, user4[1].FormattedAddress(), res.TokenPair.L2Denom)
	require.NoError(t, err)

	require.Equal(t, int64(1000), user0Balance.Int64())
	require.Equal(t, int64(1000), user1Balance.Int64())
	require.Equal(t, int64(1000), user2Balance.Int64())
	require.Equal(t, int64(1000), user3Balance.Int64())
	require.Equal(t, int64(1000), user4Balance.Int64())

	l2Amount := sdk.NewCoin(res.TokenPair.L2Denom, math.NewInt(1000))

	_, err = op.Minitia.InitiateTokenWithdrawal(ctx, user0[1].KeyName(), user1[0].FormattedAddress(), l2Amount.String(), true)
	require.NoError(t, err)
	_, err = op.Minitia.InitiateTokenWithdrawal(ctx, user1[1].KeyName(), user2[0].FormattedAddress(), l2Amount.String(), true)
	require.NoError(t, err)
	_, err = op.Minitia.InitiateTokenWithdrawal(ctx, user2[1].KeyName(), user3[0].FormattedAddress(), l2Amount.String(), true)
	require.NoError(t, err)
	_, err = op.Minitia.InitiateTokenWithdrawal(ctx, user3[1].KeyName(), user4[0].FormattedAddress(), l2Amount.String(), true)
	require.NoError(t, err)
	_, err = op.Minitia.InitiateTokenWithdrawal(ctx, user4[1].KeyName(), user0[0].FormattedAddress(), l2Amount.String(), true)
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 2, op.Initia, op.Minitia)
	require.NoError(t, err)

	_, err = op.OP.QueryWithdrawal(1)
	require.NoError(t, err)

	// Wait for the submission interval
	time.Sleep(5 * time.Second)

	w1, err := op.OP.QueryWithdrawal(1)
	require.NoError(t, err)
	w2, err := op.OP.QueryWithdrawal(2)
	require.NoError(t, err)
	w3, err := op.OP.QueryWithdrawal(3)
	require.NoError(t, err)
	w4, err := op.OP.QueryWithdrawal(4)
	require.NoError(t, err)
	w5, err := op.OP.QueryWithdrawal(5)
	require.NoError(t, err)

	err = op.Initia.WaitUntilOutputIsFinalized(ctx, 1, w5.OutputIndex)
	require.NoError(t, err)

	_, err = op.Initia.FinalizeTokenWithdrawal(ctx, user0[0].KeyName(), 1, w5.OutputIndex, w5.Sequence, w5.WithdrawalProofs, w5.From, w5.To, w5.Amount, w5.Version, w5.StorageRoot, w5.LastBlockHash, true)
	require.NoError(t, err)
	_, err = op.Initia.FinalizeTokenWithdrawal(ctx, user1[0].KeyName(), 1, w1.OutputIndex, w1.Sequence, w1.WithdrawalProofs, w1.From, w1.To, w1.Amount, w1.Version, w1.StorageRoot, w1.LastBlockHash, true)
	require.NoError(t, err)
	_, err = op.Initia.FinalizeTokenWithdrawal(ctx, user2[0].KeyName(), 1, w2.OutputIndex, w2.Sequence, w2.WithdrawalProofs, w2.From, w2.To, w2.Amount, w2.Version, w2.StorageRoot, w2.LastBlockHash, true)
	require.NoError(t, err)
	_, err = op.Initia.FinalizeTokenWithdrawal(ctx, user3[0].KeyName(), 1, w3.OutputIndex, w3.Sequence, w3.WithdrawalProofs, w3.From, w3.To, w3.Amount, w3.Version, w3.StorageRoot, w3.LastBlockHash, true)
	require.NoError(t, err)
	_, err = op.Initia.FinalizeTokenWithdrawal(ctx, user4[0].KeyName(), 1, w4.OutputIndex, w4.Sequence, w4.WithdrawalProofs, w4.From, w4.To, w4.Amount, w4.Version, w4.StorageRoot, w4.LastBlockHash, true)
	require.NoError(t, err)
}
