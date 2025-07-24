package main

import (
	"context"
	"fmt"
	"time"

	"github.com/initia-labs/opinit-bots/bot"
	bottypes "github.com/initia-labs/opinit-bots/bot/types"
	"github.com/initia-labs/opinit-bots/db"
	"github.com/initia-labs/opinit-bots/executor"
	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	"github.com/initia-labs/opinit-bots/node/rpcclient"
	"github.com/initia-labs/opinit-bots/provider/child"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/spf13/cobra"
)

// migrationCmd handles the one-time migration of withdrawal data for v0.1.5, v0.1.9
// TODO: Remove this command in the future
func migrationCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Args:  cobra.ExactArgs(1),
		Short: "Run database migrations",
		Long: `Run database migrations
v0.1.5: Store the sequence number so that it can be accessed by address
v0.1.9-1: Delete finalized trees and create new finalized trees from working trees
v0.1.9-2: Fill block hash of finalized tree 
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			version := args[0]
			switch version {
			case "v0.1.5":
				// Run migration for v0.1.5
				db, err := db.NewDB(GetDBPath(ctx.homePath, bottypes.BotTypeExecutor))
				if err != nil {
					return err
				}
				return executor.Migration015(db)
			case "v0.1.9-1":
				// Run migration for v0.1.9-1
				db, err := db.NewDB(GetDBPath(ctx.homePath, bottypes.BotTypeExecutor))
				if err != nil {
					return err
				}
				return executor.Migration019_1(db)
			case "v0.1.9-2":
				// Run migration for v0.1.9-2
				db, err := db.NewDB(GetDBPath(ctx.homePath, bottypes.BotTypeExecutor))
				if err != nil {
					return err
				}
				cmdCtx, done := context.WithCancel(cmd.Context())
				gracefulShutdown(done)
				interval, err := cmd.Flags().GetDuration(flagPollingInterval)
				if err != nil {
					return err
				}

				baseCtx := types.NewContext(cmdCtx, ctx.logger.Named(string(bottypes.BotTypeExecutor)), ctx.homePath).
					WithPollingInterval(interval)

				configPath, err := getConfigPath(cmd, ctx.homePath, string(bottypes.BotTypeExecutor))
				if err != nil {
					return err
				}

				cfg := &executortypes.Config{}
				err = bot.LoadJsonConfig(configPath, cfg)
				if err != nil {
					return err
				}

				l2Config := cfg.L2NodeConfig()
				broadcasterConfig := l2Config.BroadcasterConfig
				cdc, _, err := child.GetCodec(broadcasterConfig.Bech32Prefix)
				if err != nil {
					return err
				}

				rpcClient, err := rpcclient.NewRPCClient(cdc, l2Config.RPC, baseCtx.Logger().Named("migration-rpcclient"))
				if err != nil {
					return err
				}

				return executor.Migration019_2(baseCtx, db, rpcClient)
			case "v0.1.10":
				// Run migration for v0.1.10
				db, err := db.NewDB(GetDBPath(ctx.homePath, bottypes.BotTypeExecutor))
				if err != nil {
					return err
				}
				return executor.Migration0110(db)
			case "v0.1.11":
				// Run migration for v0.1.11
				db, err := db.NewDB(GetDBPath(ctx.homePath, bottypes.BotTypeExecutor))
				if err != nil {
					return err
				}
				return executor.Migration0111(db)
			default:
				return fmt.Errorf("unknown migration version: %s", version)
			}
		},
	}
	cmd = configFlag(ctx.v, cmd)
	cmd.Flags().Duration(flagPollingInterval, 100*time.Millisecond, "Polling interval in milliseconds")
	return cmd
}
