package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func NewRootCmd(logger *zap.Logger) *cobra.Command {
	v := viper.New()

	rootCmd := &cobra.Command{
		Use: "opbot [command]",
	}

	rootCmd.AddCommand(
		startCmd(v, logger),
	)

	return rootCmd
}
