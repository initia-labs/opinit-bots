package main

import (
	"context"
	"encoding/json"
	"fmt"

	"cosmossdk.io/errors"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"cosmossdk.io/x/feegrant"

	"github.com/cosmos/cosmos-sdk/x/authz"

	"github.com/initia-labs/opinit-bots/bot"
	bottypes "github.com/initia-labs/opinit-bots/bot/types"
	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	"github.com/initia-labs/opinit-bots/keys"
	"github.com/initia-labs/opinit-bots/node/broadcaster"
	broadcastertypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	"github.com/initia-labs/opinit-bots/node/rpcclient"
	"github.com/initia-labs/opinit-bots/provider/child"
	"github.com/initia-labs/opinit-bots/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// txCmd represents the tx command
func txCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tx",
		Short: "send a transaction",
	}

	cmd.AddCommand(
		txGrantOracleCmd(ctx),
	)
	return cmd
}

func txGrantOracleCmd(baseCtx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "grant-oracle [oracle-account-address]",
		Args:  cobra.ExactArgs(1),
		Short: "Grant oracle permission to the given account",
		Long:  `Grant oracle permission to the given account on L2 chain`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmdCtx, botDone := context.WithCancel(cmd.Context())
			gracefulShutdown(botDone)

			errGrp, ctx := errgroup.WithContext(cmdCtx)

			account, err := l2BroadcasterAccount(baseCtx, cmd)
			if err != nil {
				return err
			}

			baseCtx := types.NewContext(ctx, baseCtx.logger.Named(string(bottypes.BotTypeExecutor)), baseCtx.homePath).
				WithErrGrp(errGrp)

			err = account.Load(baseCtx)
			if err != nil {
				return err
			}

			oracleAddress, err := keys.DecodeBech32AccAddr(args[0], account.Bech32Prefix())
			if err != nil {
				return err
			}

			grantMsg, err := authz.NewMsgGrant(account.GetAddress(), oracleAddress, authz.NewGenericAuthorization(types.MsgUpdateOracleTypeUrl), nil)
			if err != nil {
				return err
			}

			msgAllowance, err := feegrant.NewAllowedMsgAllowance(&feegrant.BasicAllowance{}, []string{types.MsgUpdateOracleTypeUrl, types.MsgAuthzExecTypeUrl})
			if err != nil {
				return err
			}

			feegrantMsg, err := feegrant.NewMsgGrantAllowance(msgAllowance, account.GetAddress(), oracleAddress)
			if err != nil {
				return err
			}

			txBytes, _, err := account.BuildTxWithMsgs(ctx, []sdk.Msg{grantMsg, feegrantMsg})
			if err != nil {
				return errors.Wrapf(err, "simulation failed")
			}

			res, err := account.BroadcastTxSync(baseCtx, txBytes)
			if err != nil {
				// TODO: handle error, may repeat sending tx
				return fmt.Errorf("broadcast txs: %w", err)
			}
			bz, err := json.Marshal(res)
			if err != nil {
				return err
			}
			fmt.Println(string(bz))
			return nil
		},
	}

	cmd = configFlag(baseCtx.v, cmd)
	return cmd
}

func l2BroadcasterAccount(ctx *cmdContext, cmd *cobra.Command) (*broadcaster.BroadcasterAccount, error) {
	configPath, err := getConfigPath(cmd, ctx.homePath, string(bottypes.BotTypeExecutor))
	if err != nil {
		return nil, err
	}

	cfg := &executortypes.Config{}
	err = bot.LoadJsonConfig(configPath, cfg)
	if err != nil {
		return nil, err
	}

	l2Config := cfg.L2NodeConfig()
	broadcasterConfig := l2Config.BroadcasterConfig
	cdc, txConfig, err := child.GetCodec(broadcasterConfig.Bech32Prefix)
	if err != nil {
		return nil, err
	}

	rpcClient, err := rpcclient.NewRPCClient(cdc, l2Config.RPC)
	if err != nil {
		return nil, err
	}

	keyringConfig := broadcastertypes.KeyringConfig{
		Name: cfg.BridgeExecutor,
	}

	return broadcaster.NewBroadcasterAccount(*broadcasterConfig, cdc, txConfig, rpcClient, keyringConfig)
}
