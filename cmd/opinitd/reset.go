package main

import (
	"errors"
	"os"
	"path"

	"github.com/spf13/cobra"

	bottypes "github.com/initia-labs/opinit-bots/bot/types"
	challengerdb "github.com/initia-labs/opinit-bots/challenger/db"
	"github.com/initia-labs/opinit-bots/db"
	"github.com/initia-labs/opinit-bots/executor"
)

func resetDBCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset-db [bot-name]",
		Args:  cobra.ExactArgs(1),
		Short: "Reset a bot's db.",
		Long: `Reset a bot's db.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			botType := bottypes.BotTypeFromString(args[0])
			if err := botType.Validate(); err != nil {
				return err
			}

			dbPath := path.Join(ctx.homePath, string(botType))
			err := os.RemoveAll(dbPath + ".db")
			if err != nil {
				return err
			}

			if botType == bottypes.BotTypeExecutor {
				err = os.RemoveAll(path.Join(ctx.homePath, "batch"))
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
	return cmd
}

func resetHeightsCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset-heights [bot-name]",
		Args:  cobra.ExactArgs(1),
		Short: "Reset bot's all height info.",
		Long: `Reset bot's all height info.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			botType := bottypes.BotTypeFromString(args[0])
			if err := botType.Validate(); err != nil {
				return err
			}

			db, err := db.NewDB(GetDBPath(ctx.homePath, botType))
			if err != nil {
				return err
			}

			switch botType {
			case bottypes.BotTypeExecutor:
				return executor.ResetHeights(db)
			case bottypes.BotTypeChallenger:
				return challengerdb.ResetHeights(db)
			}
			return errors.New("unknown bot type")
		},
	}
	return cmd
}

func resetHeightCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset-height [bot-name] [node-type]",
		Args:  cobra.ExactArgs(2),
		Short: "Reset bot's node height info.",
		Long: `Reset bot's node height info.
Executor node types: 
- host
- child
- batch

Challenger node types: 
- host
- child
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			botType := bottypes.BotTypeFromString(args[0])
			if err := botType.Validate(); err != nil {
				return err
			}

			db, err := db.NewDB(GetDBPath(ctx.homePath, botType))
			if err != nil {
				return err
			}
			defer db.Close()

			switch botType {
			case bottypes.BotTypeExecutor:
				return executor.ResetHeight(db, args[1])
			case bottypes.BotTypeChallenger:
				return challengerdb.ResetHeight(db, args[1])
			}
			return errors.New("unknown bot type")
		},
	}
	return cmd
}
