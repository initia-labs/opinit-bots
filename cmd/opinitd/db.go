package main

import (
	"fmt"

	"github.com/initia-labs/opinit-bots/bot"
	bottypes "github.com/initia-labs/opinit-bots/bot/types"
	"github.com/initia-labs/opinit-bots/db"
	"github.com/initia-labs/opinit-bots/executor"
	"github.com/spf13/cobra"
)

// migration015Cmd handles the one-time migration of withdrawal data for v0.1.5
// TODO: Remove this command in the future
func migration015Cmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Args:  cobra.ExactArgs(1),
		Short: "Run database migrations",
		Long: `Run database migrations
v0.1.5: Store the sequence number so that it can be accessed by address
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			version := args[0]
			switch version {
			case "v0.1.5":
				// Run migration for v0.1.5
				db, err := db.NewDB(bot.GetDBPath(ctx.homePath, bottypes.BotTypeExecutor))
				if err != nil {
					return err
				}
				return executor.Migration015(db)
			default:
				return fmt.Errorf("unknown migration version: %s", version)
			}
		},
	}
	return cmd
}
