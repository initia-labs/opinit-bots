package main

import (
	"context"
	"os"

	"github.com/initia-labs/opinit-bots-go/cmd"
)

// TODO: use cmd package to build and run the bot
// just test the bot with this main function

func main() {
	rootCmd := cmd.NewRootCmd()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
