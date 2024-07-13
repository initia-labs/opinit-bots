package bot

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/initia-labs/OPinit/x/opchild"
	"github.com/initia-labs/OPinit/x/ophost"
	initiaapp "github.com/initia-labs/initia/app"
	"github.com/initia-labs/initia/app/params"
	bottypes "github.com/initia-labs/opinit-bots-go/bot/types"
	"github.com/initia-labs/opinit-bots-go/db"
	"github.com/initia-labs/opinit-bots-go/executor"
	executortypes "github.com/initia-labs/opinit-bots-go/executor/types"
	"github.com/initia-labs/opinit-bots-go/server"
	"go.uber.org/zap"
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

func NewBot(name string, logger *zap.Logger, homePath string, configPath string) (bottypes.Bot, error) {
	SetSDKConfig()

	encodingConfig := params.MakeEncodingConfig()
	appCodec := encodingConfig.Codec
	txConfig := encodingConfig.TxConfig

	std.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	std.RegisterLegacyAminoCodec(encodingConfig.Amino)
	auth.AppModuleBasic{}.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	ophost.AppModuleBasic{}.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	opchild.AppModuleBasic{}.RegisterInterfaces(encodingConfig.InterfaceRegistry)

	auth.AppModuleBasic{}.RegisterLegacyAminoCodec(encodingConfig.Amino)
	ophost.AppModuleBasic{}.RegisterLegacyAminoCodec(encodingConfig.Amino)
	opchild.AppModuleBasic{}.RegisterLegacyAminoCodec(encodingConfig.Amino)

	switch name {
	case bottypes.ExecutorName:
		cfg := &executortypes.Config{}
		err := LoadJsonConfig(configPath, cfg)
		if err != nil {
			return nil, err
		}
		db, err := db.NewDB(getDBPath(homePath, name))
		if err != nil {
			return nil, err
		}
		server := server.NewServer()
		return executor.NewExecutor(cfg, db, server, logger, appCodec, txConfig), nil
	}

	return nil, errors.New("not providing bot name")
}

func SetSDKConfig() {
	sdkConfig := sdk.GetConfig()
	sdkConfig.SetCoinType(initiaapp.CoinType)

	accountPubKeyPrefix := initiaapp.AccountAddressPrefix + "pub"
	validatorAddressPrefix := initiaapp.AccountAddressPrefix + "valoper"
	validatorPubKeyPrefix := initiaapp.AccountAddressPrefix + "valoperpub"
	consNodeAddressPrefix := initiaapp.AccountAddressPrefix + "valcons"
	consNodePubKeyPrefix := initiaapp.AccountAddressPrefix + "valconspub"

	sdkConfig.SetBech32PrefixForAccount(initiaapp.AccountAddressPrefix, accountPubKeyPrefix)
	sdkConfig.SetBech32PrefixForValidator(validatorAddressPrefix, validatorPubKeyPrefix)
	sdkConfig.SetBech32PrefixForConsensusNode(consNodeAddressPrefix, consNodePubKeyPrefix)
	sdkConfig.SetAddressVerifier(initiaapp.VerifyAddressLen())
	sdkConfig.Seal()
}

func getDBPath(homePath string, botName string) string {
	return fmt.Sprintf(homePath+"/%s.db", botName)
}
