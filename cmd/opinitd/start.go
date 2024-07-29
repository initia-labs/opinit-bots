package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/initia-labs/opinit-bots-go/bot"
	bottypes "github.com/initia-labs/opinit-bots-go/bot/types"
)

func startCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start [bot-name]",
		Args:  cobra.ExactArgs(1),
		Short: "Start a bot with the given name",
		Long: `Start a bot with the given name. 

Currently supported bots: 
- executor
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configName, err := cmd.Flags().GetString(flagConfigName)
			if err != nil {
				return err
			}

			botType := bottypes.BotTypeFromString(args[0])
			bot, err := bot.NewBot(botType, ctx.logger, ctx.homePath, configName)
			if err != nil {
				return err
			}

			cmdCtx, botDone := context.WithCancel(cmd.Context())
			gracefulShutdown(botDone)

			return bot.Start(cmdCtx)
		},
	}

	cmd = configFlag(ctx.v, cmd)
	return cmd
}

func gracefulShutdown(done context.CancelFunc) {
	signalChannel := make(chan os.Signal, 2)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalChannel
		done()
	}()
}