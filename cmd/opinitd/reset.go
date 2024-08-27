package main

import (
	"os"
	"path"

	"github.com/spf13/cobra"

	bottypes "github.com/initia-labs/opinit-bots/bot/types"
)

func resetDBCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset-db [bot-name]",
		Args:  cobra.ExactArgs(1),
		Short: "Reset a bot's db.",
		Long: `Reset a bot's db.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			botType := bottypes.BotTypeFromString(args[0])
			switch botType {
			case bottypes.BotTypeExecutor:
				dbPath := path.Join(ctx.homePath, string(botType))
				err := os.RemoveAll(dbPath + ".db")
				if err != nil {
					return err
				}
				err = os.RemoveAll(path.Join(ctx.homePath, "batch"))
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
	return cmd
}
