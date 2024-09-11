package main

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	flagHome       = "home"
	flagConfigName = "config"
)

var defaultHome = filepath.Join(os.Getenv("HOME"), ".opinit")

func configFlag(v *viper.Viper, cmd *cobra.Command) *cobra.Command {
	cmd.Flags().StringP(flagConfigName, "c", "", "The name of the configuration file in the home directory. Must have json extension. default: executor.json for executor, challenger.json for challenger")
	if err := v.BindPFlag(flagConfigName, cmd.Flags().Lookup(flagConfigName)); err != nil {
		panic(err)
	}

	return cmd
}

func getConfigPath(cmd *cobra.Command, homePath, botName string) (string, error) {
	configName, err := cmd.Flags().GetString(flagConfigName)
	if err != nil {
		return "", err
	}
	if configName == "" {
		configName = fmt.Sprintf("%s.json", botName)
	}
	configPath := path.Join(homePath, configName)
	if path.Ext(configPath) != ".json" {
		return "", errors.New("config file must be a json file")
	}
	return configPath, nil
}
