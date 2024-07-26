package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	flagHome            = "home"
	flagConfigName      = "config"
	flagExecutorKeyName = "executor"
)

func configFlag(v *viper.Viper, cmd *cobra.Command) *cobra.Command {
	cmd.Flags().StringP(flagConfigName, "c", "executor.json", "The name of the configuration file in the home directory")
	if err := v.BindPFlag(flagConfigName, cmd.Flags().Lookup(flagConfigName)); err != nil {
		panic(err)
	}

	return cmd
}
