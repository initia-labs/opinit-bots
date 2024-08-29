package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/initia-labs/opinit-bots/bot"
	bottypes "github.com/initia-labs/opinit-bots/bot/types"
	"github.com/initia-labs/opinit-bots/types"
)

const (
	flagPollingInterval = "polling-interval"
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
			botType := bottypes.BotTypeFromString(args[0])
			if err := botType.Validate(); err != nil {
				return err
			}

			configPath, err := getConfigPath(cmd, ctx.homePath, args[0])
			if err != nil {
				return err
			}

			bot, err := bot.NewBot(botType, ctx.logger, ctx.homePath, configPath)
			if err != nil {
				return err
			}

			cmdCtx, botDone := context.WithCancel(cmd.Context())
			gracefulShutdown(botDone)

			errGrp, ctx := errgroup.WithContext(cmdCtx)
			ctx = types.WithErrGrp(ctx, errGrp)
			interval, err := cmd.Flags().GetDuration(flagPollingInterval)
			if err != nil {
				return err
			}
			ctx = types.WithPollingInterval(ctx, interval)
			err = bot.Initialize(ctx)
			if err != nil {
				return err
			}
			return bot.Start(ctx)
		},
	}

	cmd = configFlag(ctx.v, cmd)
	cmd.Flags().Duration(flagPollingInterval, 100*time.Millisecond, "Polling interval in milliseconds")
	return cmd
}

func gracefulShutdown(done context.CancelFunc) {
	signalChannel := make(chan os.Signal, 2)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalChannel
		fmt.Println("Received signal to stop. Shutting down...")
		done()
	}()
}
