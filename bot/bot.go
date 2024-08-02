package bot

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"

	"go.uber.org/zap"

	bottypes "github.com/initia-labs/opinit-bots-go/bot/types"
	"github.com/initia-labs/opinit-bots-go/db"
	"github.com/initia-labs/opinit-bots-go/executor"
	executortypes "github.com/initia-labs/opinit-bots-go/executor/types"
	"github.com/initia-labs/opinit-bots-go/server"
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

// func SetSDKConfig() {
// 	sdkConfig := sdk.GetConfig()
// 	sdkConfig.SetCoinType(initiaapp.CoinType)

// 	accountPubKeyPrefix := initiaapp.AccountAddressPrefix + "pub"
// 	validatorAddressPrefix := initiaapp.AccountAddressPrefix + "valoper"
// 	validatorPubKeyPrefix := initiaapp.AccountAddressPrefix + "valoperpub"
// 	consNodeAddressPrefix := initiaapp.AccountAddressPrefix + "valcons"
// 	consNodePubKeyPrefix := initiaapp.AccountAddressPrefix + "valconspub"

// 	sdkConfig.SetBech32PrefixForAccount(initiaapp.AccountAddressPrefix, accountPubKeyPrefix)
// 	sdkConfig.SetBech32PrefixForValidator(validatorAddressPrefix, validatorPubKeyPrefix)
// 	sdkConfig.SetBech32PrefixForConsensusNode(consNodeAddressPrefix, consNodePubKeyPrefix)
// 	sdkConfig.SetAddressVerifier(initiaapp.VerifyAddressLen())
// 	sdkConfig.Seal()
// }

func getDBPath(homePath string, botName bottypes.BotType) string {
	return fmt.Sprintf(homePath+"/%s.db", botName)
}
