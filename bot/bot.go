package bot

import (
	"encoding/json"
	"errors"
	"io"
	"os"

	bottypes "github.com/initia-labs/opinit-bots/bot/types"
	"github.com/initia-labs/opinit-bots/challenger"
	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	"github.com/initia-labs/opinit-bots/executor"
	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	"github.com/initia-labs/opinit-bots/server"
	"github.com/initia-labs/opinit-bots/types"
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

func NewBot(botType bottypes.BotType, db types.DB, configPath string) (bottypes.Bot, error) {
	err := botType.Validate()
	if err != nil {
		return nil, err
	}

	switch botType {
	case bottypes.BotTypeExecutor:
		cfg := &executortypes.Config{}
		err := LoadJsonConfig(configPath, cfg)
		if err != nil {
			return nil, err
		}
		server := server.NewServer(cfg.Server)
		return executor.NewExecutor(cfg, db, server), nil
	case bottypes.BotTypeChallenger:
		cfg := &challengertypes.Config{}
		err := LoadJsonConfig(configPath, cfg)
		if err != nil {
			return nil, err
		}
		server := server.NewServer(cfg.Server)
		return challenger.NewChallenger(cfg, db, server), nil
	}
	return nil, errors.New("not providing bot name")
}
