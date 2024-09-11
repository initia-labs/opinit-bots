package main

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"

	bottypes "github.com/initia-labs/opinit-bots/bot/types"
	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	executortypes "github.com/initia-labs/opinit-bots/executor/types"
)

func initCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [bot-name]",
		Args:  cobra.ExactArgs(1),
		Short: "Initialize a bot's configuration files.",
		Long: `Initialize a bot's configuration files.

Currently supported bots are: executor
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

			if err := os.MkdirAll(ctx.homePath, os.ModePerm); err != nil {
				return err
			}

			f, err := os.Create(configPath)
			if err != nil {
				return err
			}

			var config interface{}
			switch botType {
			case bottypes.BotTypeExecutor:
				config = executortypes.DefaultConfig()
			case bottypes.BotTypeChallenger:
				config = challengertypes.DefaultConfig()
			}

			bz, err := json.MarshalIndent(config, "", "  ")
			if err != nil {
				return err
			}

			if _, err := f.Write(bz); err != nil {
				return err
			}

			return nil
		},
	}

	cmd = configFlag(ctx.v, cmd)
	return cmd
}
