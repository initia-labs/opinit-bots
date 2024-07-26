package cmd

import (
	"encoding/json"
	"errors"
	"os"
	"path"

	"github.com/spf13/cobra"

	bottypes "github.com/initia-labs/opinit-bots-go/bot/types"
	executortypes "github.com/initia-labs/opinit-bots-go/executor/types"
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
			configName, err := cmd.Flags().GetString(flagConfigName)
			if err != nil {
				return err
			}

			configPath := path.Join(ctx.homePath, configName)
			if path.Ext(configPath) != ".json" {
				return errors.New("config file must be a json file")
			}

			botType := bottypes.BotTypeFromString(args[0])
			switch botType {
			case bottypes.BotTypeExecutor:
				f, err := os.Create(configPath)
				if err != nil {
					return err
				}

				bz, err := json.MarshalIndent(executortypes.DefaultConfig(), "", "  ")
				if err != nil {
					return err
				}

				if _, err := f.Write(bz); err != nil {
					return err
				}
			}

			return nil
		},
	}

	cmd = configFlag(ctx.v, cmd)
	return cmd
}
