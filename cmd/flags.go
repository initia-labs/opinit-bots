package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	flagHome            = "home"
	flagConfigPath      = "config"
	flagExecutorKeyName = "executor"
)

func configFlag(v *viper.Viper, cmd *cobra.Command) *cobra.Command {
	cmd.Flags().StringP(flagConfigPath, "c", "", "config file path")
	if err := v.BindPFlag(flagConfigPath, cmd.Flags().Lookup(flagConfigPath)); err != nil {
		panic(err)
	}

	return cmd
}
