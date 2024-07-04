package bot

import (
	"encoding/json"
	"errors"
	"io"
	"os"

	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	initiaapp "github.com/initia-labs/initia/app"
	"github.com/initia-labs/initia/app/params"
	"github.com/initia-labs/opinit-bots-go/bot/types"
	"github.com/initia-labs/opinit-bots-go/db"
	"github.com/initia-labs/opinit-bots-go/executor"
	executortypes "github.com/initia-labs/opinit-bots-go/executor/types"
	"go.uber.org/zap"
)

func LoadJsonConfig(path string, config types.Config) error {
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

func NewBot(name string, logger *zap.Logger, homePath string, configPath string) (types.Bot, error) {
	SetSDKConfig()

	encodingConfig := params.MakeEncodingConfig()
	appCodec := encodingConfig.Codec
	txConfig := encodingConfig.TxConfig

	std.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	std.RegisterLegacyAminoCodec(encodingConfig.Amino)
	auth.AppModuleBasic{}.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	auth.AppModuleBasic{}.RegisterLegacyAminoCodec(encodingConfig.Amino)

	switch name {
	case "executor":
		cfg := &executortypes.Config{}
		err := LoadJsonConfig(configPath, cfg)
		if err != nil {
			return nil, err
		}
		db, err := db.NewDB(homePath)
		if err != nil {
			return nil, err
		}
		return executor.NewExecutor(cfg, db.WithPrefix([]byte(name)), logger, appCodec, txConfig), nil
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
