package main

import (
	"fmt"
	"os"
)

// TODO: use cmd package to build and run the bot
// just test the bot with this main function

func main() {
	rootCmd := NewRootCmd()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(rootCmd.OutOrStderr(), err)
		os.Exit(1)
	}
}
