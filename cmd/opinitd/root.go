package main

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/initia-labs/opinit-bots/version"
)

func NewRootCmd() *cobra.Command {
	ctx := &cmdContext{
		v: viper.New(),
	}

	rootCmd := &cobra.Command{
		Use: "opinitd [command]",
	}

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) (err error) {
		ctx.logger, err = getLogger(ctx.v.GetString("log-level"))
		if err != nil {
			return err
		}
		return nil
	}

	rootCmd.PersistentPostRun = func(cmd *cobra.Command, _ []string) {
		_ = ctx.logger.Sync()
	}

	rootCmd.PersistentFlags().StringVar(&ctx.homePath, flagHome, defaultHome, "set home directory")
	if err := ctx.v.BindPFlag(flagHome, rootCmd.PersistentFlags().Lookup(flagHome)); err != nil {
		panic(err)
	}

	rootCmd.PersistentFlags().String("log-level", "", "log level format (info, debug, warn, error, panic or fatal)")
	if err := ctx.v.BindPFlag("log-level", rootCmd.PersistentFlags().Lookup("log-level")); err != nil {
		panic(err)
	}

	rootCmd.AddCommand(
		initCmd(ctx),
		startCmd(ctx),
		keysCmd(ctx),
		resetDBCmd(ctx),
		resetHeightsCmd(ctx),
		resetHeightCmd(ctx),
		version.NewVersionCommand(),
	)
	return rootCmd
}

func getLogger(logLevel string) (*zap.Logger, error) {
	level := zap.InfoLevel
	switch logLevel {
	case "debug":
		level = zap.DebugLevel
	case "warn":
		level = zap.WarnLevel
	case "error":
		level = zap.ErrorLevel
	case "panic":
		level = zap.PanicLevel
	case "fatal":
		level = zap.FatalLevel
	}

	config := zap.NewDevelopmentConfig()
	config.Level = zap.NewAtomicLevelAt(level)
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	return config.Build()
}
