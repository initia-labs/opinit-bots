package cmd

import (
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type cmdContext struct {
	v        *viper.Viper
	logger   *zap.Logger
	homePath string
}
