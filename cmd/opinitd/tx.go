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

			existingGrants, err := queryAuthzGrants(baseCtx, cfg, account.GetAddressString(), oracleAddress.String())
			if err != nil {
				return errors.Wrap(err, "failed to query authz grants")
			}

			msgs := make([]sdk.Msg, 0)

			if !hasAuthzGrant(existingGrants, types.MsgUpdateOracleTypeUrl) {
				grantUpdateMsg, err := authz.NewMsgGrant(account.GetAddress(), oracleAddress, authz.NewGenericAuthorization(types.MsgUpdateOracleTypeUrl), nil)
				if err != nil {
					return err
				}
				msgs = append(msgs, grantUpdateMsg)
				fmt.Println("Adding authz grant for MsgUpdateOracle")
			} else {
				fmt.Println("MsgUpdateOracle authz grant already exists, skipping")
			}

			if !hasAuthzGrant(existingGrants, types.MsgRelayOracleTypeUrl) {
				grantRelayMsg, err := authz.NewMsgGrant(account.GetAddress(), oracleAddress, authz.NewGenericAuthorization(types.MsgRelayOracleTypeUrl), nil)
				if err != nil {
					return err
				}
				msgs = append(msgs, grantRelayMsg)
				fmt.Println("Adding authz grant for MsgRelayOracleData")
			} else {
				fmt.Println("MsgRelayOracleData authz grant already exists, skipping")
			}

			existingAllowance, err := queryFeegrant(baseCtx, cfg, account.GetAddressString(), oracleAddress.String())
			if err != nil {
				return errors.Wrap(err, "failed to query feegrant")
			}

			requiredMsgTypes := []string{types.MsgRelayOracleTypeUrl, types.MsgUpdateOracleTypeUrl, types.MsgAuthzExecTypeUrl}

			if existingAllowance != nil {
				hasAllTypes, err := hasAllMsgTypes(existingAllowance, requiredMsgTypes)
				if err != nil {
					return err
				}

				if !hasAllTypes {
					revokeMsg := feegrant.NewMsgRevokeAllowance(account.GetAddress(), oracleAddress)
					msgs = append(msgs, &revokeMsg)
					fmt.Println("Revoking existing feegrant to update with complete message types")

					msgAllowance, err := feegrant.NewAllowedMsgAllowance(&feegrant.BasicAllowance{}, requiredMsgTypes)
					if err != nil {
						return err
					}

					feegrantMsg, err := feegrant.NewMsgGrantAllowance(msgAllowance, account.GetAddress(), oracleAddress)
					if err != nil {
						return err
					}

					msgs = append(msgs, feegrantMsg)
					fmt.Println("Adding feegrant with all required message types")
				} else {
					fmt.Println("Feegrant already exists with all required message types, skipping")
				}
			} else {
				msgAllowance, err := feegrant.NewAllowedMsgAllowance(&feegrant.BasicAllowance{}, requiredMsgTypes)
				if err != nil {
					return err
				}

				feegrantMsg, err := feegrant.NewMsgGrantAllowance(msgAllowance, account.GetAddress(), oracleAddress)
				if err != nil {
					return err
				}

				msgs = append(msgs, feegrantMsg)
				fmt.Println("Adding new feegrant")
			}

			if len(msgs) == 0 {
				fmt.Println("All grants already exist, nothing to do")
				return nil
			}

			txBytes, _, err := account.BuildTxWithMsgs(ctx, msgs)
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

	l2RpcClient, err := rpcclient.NewRPCClient(cdc, l2Config.RPC)
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

	rpcClient, err := rpcclient.NewRPCClient(cdc, l1Config.RPC)
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

	rpcClient, err := rpcclient.NewRPCClient(cdc, l2Config.RPC)
	if err != nil {
		return nil, err
	}

	keyringConfig := broadcastertypes.KeyringConfig{
		Name: cfg.BridgeExecutor,
	}

	return broadcaster.NewBroadcasterAccount(ctx, *broadcasterConfig, cdc, txConfig, rpcClient, keyringConfig)
}

// queryAuthzGrants queries existing authz grants from granter to grantee
func queryAuthzGrants(ctx types.Context, cfg *executortypes.Config, granter string, grantee string) ([]*authz.Grant, error) {
	l2Config := cfg.L2NodeConfig()
	cdc, _, err := child.GetCodec(l2Config.BroadcasterConfig.Bech32Prefix)
	if err != nil {
		return nil, err
	}

	rpcClient, err := rpcclient.NewRPCClient(cdc, l2Config.RPC)
	if err != nil {
		return nil, err
	}
	queryCtx, cancel := rpcclient.GetQueryContext(ctx, 0)
	defer cancel()
	resp, err := authz.NewQueryClient(rpcClient).Grants(queryCtx, &authz.QueryGrantsRequest{
		Granter: granter,
		Grantee: grantee,
	})
	if err != nil {
		return nil, err
	}

	for _, grant := range resp.Grants {
		if grant.Authorization != nil {
			var authI authz.Authorization
			err = cdc.UnpackAny(grant.Authorization, &authI)
			if err != nil {
				return nil, errors.Wrap(err, "failed to unpack authorization")
			}
		}
	}

	return resp.Grants, nil
}

// hasAuthzGrant checks if a grant exists for the given message type URL
func hasAuthzGrant(grants []*authz.Grant, msgTypeUrl string) bool {
	for _, grant := range grants {
		auth, ok := grant.Authorization.GetCachedValue().(*authz.GenericAuthorization)
		if ok && auth.Msg == msgTypeUrl {
			return true
		}
	}

	return false
}

// queryFeegrant queries existing feegrant allowance from granter to grantee
func queryFeegrant(ctx types.Context, cfg *executortypes.Config, granter string, grantee string) (*feegrant.Grant, error) {
	l2Config := cfg.L2NodeConfig()
	cdc, _, err := child.GetCodec(l2Config.BroadcasterConfig.Bech32Prefix)
	if err != nil {
		return nil, err
	}

	rpcClient, err := rpcclient.NewRPCClient(cdc, l2Config.RPC)
	if err != nil {
		return nil, err
	}
	queryCtx, cancel := rpcclient.GetQueryContext(ctx, 0)
	defer cancel()
	resp, err := feegrant.NewQueryClient(rpcClient).Allowance(queryCtx, &feegrant.QueryAllowanceRequest{
		Granter: granter,
		Grantee: grantee,
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "no allowance") {
			return nil, nil
		}
		return nil, err
	}

	if resp.Allowance != nil && resp.Allowance.Allowance != nil {
		var allowanceI feegrant.FeeAllowanceI
		err = cdc.UnpackAny(resp.Allowance.Allowance, &allowanceI)
		if err != nil {
			return nil, errors.Wrap(err, "failed to unpack allowance")
		}
	}

	return resp.Allowance, nil
}

// hasAllMsgTypes checks if the feegrant allowance includes all required message types
func hasAllMsgTypes(grant *feegrant.Grant, requiredMsgTypes []string) (bool, error) {
	if grant == nil {
		return false, nil
	}

	allowance := grant.Allowance.GetCachedValue()
	if allowance == nil {
		return false, fmt.Errorf("allowance cached value is nil")
	}

	switch a := allowance.(type) {
	case *feegrant.AllowedMsgAllowance:
		allowedMsgs := a.AllowedMessages
		if len(allowedMsgs) == 0 {
			return true, nil
		}

		allowedSet := make(map[string]bool)
		for _, msg := range allowedMsgs {
			allowedSet[msg] = true
		}

		for _, required := range requiredMsgTypes {
			if !allowedSet[required] {
				return false, nil
			}
		}

		return true, nil
	default:
		// other types, assume they need to be updated
		return false, nil
	}
}
