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

	"github.com/initia-labs/opinit-bots-go/bot"
	bottypes "github.com/initia-labs/opinit-bots-go/bot/types"
	"github.com/initia-labs/opinit-bots-go/types"
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
			errGrp, ctx := errgroup.WithContext(cmdCtx)
			ctx = types.WithErrGrp(ctx, errGrp)
			interval, err := cmd.Flags().GetDuration(flagPollingInterval)
			ctx = types.WithPollingInterval(ctx, interval)

			err = bot.Initialize(ctx)
			if err != nil {
				return err
			}

			gracefulShutdown(botDone)
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
