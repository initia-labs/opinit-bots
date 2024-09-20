package bot

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"go.uber.org/zap"

	bottypes "github.com/initia-labs/opinit-bots/bot/types"
	"github.com/initia-labs/opinit-bots/challenger"
	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
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

func NewBot(botType bottypes.BotType, logger *zap.Logger, homePath string, configPath string) (bottypes.Bot, error) {
	err := botType.Validate()
	if err != nil {
		return nil, err
	}

	db, err := db.NewDB(GetDBPath(homePath, botType))
	if err != nil {
		return nil, err
	}
	server := server.NewServer()

	switch botType {
	case bottypes.BotTypeExecutor:
		cfg := &executortypes.Config{}
		err := LoadJsonConfig(configPath, cfg)
		if err != nil {
			return nil, err
		}
		return executor.NewExecutor(cfg, db, server, logger.Named("executor"), homePath), nil
	case bottypes.BotTypeChallenger:
		cfg := &challengertypes.Config{}
		err := LoadJsonConfig(configPath, cfg)
		if err != nil {
			return nil, err
		}
		return challenger.NewChallenger(cfg, db, server, logger.Named("challenger"), homePath), nil
	}
	return nil, errors.New("not providing bot name")
}

func GetDBPath(homePath string, botName bottypes.BotType) string {
	return fmt.Sprintf(homePath+"/%s.db", botName)
}
