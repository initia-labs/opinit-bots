package bot

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"

	"go.uber.org/zap"

	bottypes "github.com/initia-labs/opinit-bots/bot/types"
	"github.com/initia-labs/opinit-bots/db"
	"github.com/initia-labs/opinit-bots/executor"
	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	"github.com/initia-labs/opinit-bots/server"
)

func LoadJsonConfig(path string, config bottypes.Config) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, config)
	if err != nil {
		return err
	}

	return nil
}

func NewBot(name bottypes.BotType, logger *zap.Logger, homePath string, configName string) (bottypes.Bot, error) {
	switch name {
	case bottypes.BotTypeExecutor:
		cfg := &executortypes.Config{}

		configPath := path.Join(homePath, configName)
		err := LoadJsonConfig(configPath, cfg)
		if err != nil {
			return nil, err
		}
		db, err := db.NewDB(getDBPath(homePath, name))
		if err != nil {
			return nil, err
		}
		server := server.NewServer()
		return executor.NewExecutor(cfg, db, server, logger.Named("executor"), homePath), nil
	}

	return nil, errors.New("not providing bot name")
}

func getDBPath(homePath string, botName bottypes.BotType) string {
	return fmt.Sprintf(homePath+"/%s.db", botName)
}
