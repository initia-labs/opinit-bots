package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

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
		txUpdateBatchInfoCmd(ctx),
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

			baseCtx := types.NewContext(ctx, baseCtx.logger.Named(string(bottypes.BotTypeExecutor)), baseCtx.homePath).
				WithErrGrp(errGrp)

			cfg, err := getExecutorConfig(baseCtx, cmd)
			if err != nil {
				return err
			}

			account, err := l2BroadcasterAccount(baseCtx, cfg)
			if err != nil {
				return err
			}

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

func txUpdateBatchInfoCmd(baseCtx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-batch-info [chain-type] [new-submitter-address]",
		Args:  cobra.ExactArgs(2),
		Short: "Update batch info with new chain type and new submitter address",
		Long: `Update batch info with new chain type and new submitter address.
Before running this command, you need to 
(1) register a new key for the new submitter address
(2) update the DA configuration in the executor config.
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmdCtx, botDone := context.WithCancel(cmd.Context())
			gracefulShutdown(botDone)

			errGrp, ctx := errgroup.WithContext(cmdCtx)

			baseCtx := types.NewContext(ctx, baseCtx.logger.Named(string(bottypes.BotTypeExecutor)), baseCtx.homePath).
				WithErrGrp(errGrp)

			cfg, err := getExecutorConfig(baseCtx, cmd)
			if err != nil {
				return err
			}

			chainType := args[0]
			newSubmitterAddress := args[1]

			err = validateBatchInfoArgs(chainType)
			if err != nil {
				return err
			}

			bridgeId, err := QueryBridgeId(baseCtx, cfg)
			if err != nil {
				return err
			}

			proposerAccount, err := l1ProposerAccount(baseCtx, cfg, bridgeId)
			if err != nil {
				return err
			}

			err = proposerAccount.Load(baseCtx)
			if err != nil {
				return err
			}

			updateBatchInfoMsg := &ophosttypes.MsgUpdateBatchInfo{
				Authority: proposerAccount.GetAddressString(),
				BridgeId:  bridgeId,
				NewBatchInfo: ophosttypes.BatchInfo{
					Submitter: newSubmitterAddress,
					ChainType: ophosttypes.BatchInfo_ChainType(ophosttypes.BatchInfo_ChainType_value[chainType]),
				},
			}

			txBytes, _, err := proposerAccount.BuildTxWithMsgs(ctx, []sdk.Msg{updateBatchInfoMsg})
			if err != nil {
				return errors.Wrapf(err, "simulation failed")
			}

			res, err := proposerAccount.BroadcastTxSync(baseCtx, txBytes)
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

func validateBatchInfoArgs(chainType string) error {
	chainType = strings.ToUpper(chainType)
	if chainType != ophosttypes.BatchInfo_INITIA.String() && chainType != ophosttypes.BatchInfo_CELESTIA.String() {
		return fmt.Errorf("supported chain type: %s, %s", ophosttypes.BatchInfo_INITIA.String(), ophosttypes.BatchInfo_CELESTIA.String())
	}
	return nil
}

func getExecutorConfig(ctx types.Context, cmd *cobra.Command) (*executortypes.Config, error) {
	configPath, err := getConfigPath(cmd, ctx.HomePath(), string(bottypes.BotTypeExecutor))
	if err != nil {
		return nil, err
	}

	cfg := &executortypes.Config{}
	err = bot.LoadJsonConfig(configPath, cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func QueryBridgeId(ctx types.Context, cfg *executortypes.Config) (uint64, error) {
	l2Config := cfg.L2NodeConfig()
	cdc, _, err := child.GetCodec(l2Config.BroadcasterConfig.Bech32Prefix)
	if err != nil {
		return 0, err
	}

	l2RpcClient, err := rpcclient.NewRPCClient(ctx, cdc, l2Config.RPC, ctx.Logger().Named("l2-rpcclient"))
	if err != nil {
		return 0, err
	}
	queryCtx, cancel := rpcclient.GetQueryContext(ctx, 0)
	defer cancel()
	bridgeInfoResponse, err := opchildtypes.NewQueryClient(l2RpcClient).BridgeInfo(queryCtx, &opchildtypes.QueryBridgeInfoRequest{})
	if err != nil {
		return 0, errors.Wrap(err, "failed to query opchild bridge info")
	}
	return bridgeInfoResponse.BridgeInfo.BridgeId, nil
}

func l1ProposerAccount(ctx types.Context, cfg *executortypes.Config, bridgeId uint64) (*broadcaster.BroadcasterAccount, error) {
	l1Config := cfg.L1NodeConfig()
	broadcasterConfig := l1Config.BroadcasterConfig
	cdc, txConfig, err := child.GetCodec(broadcasterConfig.Bech32Prefix)
	if err != nil {
		return nil, err
	}

	rpcClient, err := rpcclient.NewRPCClient(ctx, cdc, l1Config.RPC, ctx.Logger().Named("l1-rpcclient"))
	if err != nil {
		return nil, err
	}
	queryCtx, cancel := rpcclient.GetQueryContext(ctx, 0)
	defer cancel()
	bridgeResponse, err := ophosttypes.NewQueryClient(rpcClient).Bridge(queryCtx, &ophosttypes.QueryBridgeRequest{BridgeId: bridgeId})
	if err != nil {
		return nil, errors.Wrap(err, "failed to query ophost bridge info")
	}

	keyringConfig := broadcastertypes.KeyringConfig{
		Address: bridgeResponse.BridgeConfig.Proposer,
	}
	return broadcaster.NewBroadcasterAccount(ctx, *broadcasterConfig, cdc, txConfig, rpcClient, keyringConfig)
}

func l2BroadcasterAccount(ctx types.Context, cfg *executortypes.Config) (*broadcaster.BroadcasterAccount, error) {
	l2Config := cfg.L2NodeConfig()
	broadcasterConfig := l2Config.BroadcasterConfig
	cdc, txConfig, err := child.GetCodec(broadcasterConfig.Bech32Prefix)
	if err != nil {
		return nil, err
	}

	rpcClient, err := rpcclient.NewRPCClient(ctx, cdc, l2Config.RPC, ctx.Logger().Named("l2-rpcclient"))
	if err != nil {
		return nil, err
	}

	keyringConfig := broadcastertypes.KeyringConfig{
		Name: cfg.BridgeExecutor,
	}

	return broadcaster.NewBroadcasterAccount(ctx, *broadcasterConfig, cdc, txConfig, rpcClient, keyringConfig)
}
