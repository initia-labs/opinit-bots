package main

import (
	"fmt"

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
			case "v0.1.6":
				// Run migration for v0.1.5
				db, err := db.NewDB(GetDBPath(ctx.homePath, bottypes.BotTypeExecutor))
				if err != nil {
					return err
				}
				defer db.Close()
				return executor.Migration016(db)
			default:
				return fmt.Errorf("unknown migration version: %s", version)
			}
		},
	}
	return cmd
}
