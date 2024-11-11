package main

import (
	"github.com/initia-labs/opinit-bots/bot"
	bottypes "github.com/initia-labs/opinit-bots/bot/types"
	"github.com/initia-labs/opinit-bots/db"
	"github.com/initia-labs/opinit-bots/executor"
	"github.com/spf13/cobra"
)

// temporary db migration command for v0.1.5
func Migration015Cmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migration",
		Args:  cobra.ExactArgs(0),
		Short: "Migration command for v0.1.5",
		Long: `Migration command for v0.1.5
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := db.NewDB(bot.GetDBPath(ctx.homePath, bottypes.BotTypeExecutor))
			if err != nil {
				return err
			}

			return executor.Migration015(db)
		},
	}
	return cmd
}
