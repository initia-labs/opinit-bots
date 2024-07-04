package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewRootCmd() *cobra.Command {
	cmdContext := &cmdContext{
		v: viper.New(),
	}

	rootCmd := &cobra.Command{
		Use: "opbot [command]",
	}

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) (err error) {
		cmdContext.logger, err = getLogger(cmdContext.v.GetString("log-level"))
		if err != nil {
			return err
		}
		return nil
	}

	rootCmd.PersistentPostRun = func(cmd *cobra.Command, _ []string) {
		_ = cmdContext.logger.Sync()
	}

	rootCmd.PersistentFlags().String("log-level", "", "log level format (info, debug, warn, error, panic or fatal)")
	if err := cmdContext.v.BindPFlag("log-level", rootCmd.PersistentFlags().Lookup("log-level")); err != nil {
		panic(err)
	}

	rootCmd.AddCommand(
		startCmd(cmdContext),
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
