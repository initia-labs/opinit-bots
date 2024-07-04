package cmd

import (
	"github.com/spf13/cobra"

	bot "github.com/initia-labs/opinit-bots-go/bot"
)

func startCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:  "start [bot-name]",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := cmd.Flags().GetString(flagConfigPath)
			if err != nil {
				return err
			}

			bot, err := bot.NewBot(args[0], configPath, ctx.logger)
			if err != nil {
				return err
			}
			return bot.Start(cmd.Context())
		},
	}

	cmd = configFlag(ctx.v, cmd)
	return cmd
}
