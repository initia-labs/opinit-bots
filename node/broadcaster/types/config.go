package types

import (
	"context"
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/initia-labs/opinit-bots-go/keys"
)

type BuildTxWithMessagesFn func(context.Context, []sdk.Msg) ([]byte, string, error)
type PendingTxToProcessedMsgsFn func([]byte) ([]sdk.Msg, error)

type BroadcasterConfig struct {
	// ChainID is the chain ID.
	ChainID string

	// GasPrice is the gas price.
	GasPrice string

	// GasAdjustment is the gas adjustment.
	GasAdjustment float64

	// TxTimeout is the transaction timeout.
	TxTimeout time.Duration

	// Bech32Prefix is the Bech32 prefix.
	Bech32Prefix string

	// BuildTxWithMessages is the function to build a transaction with messages.
	BuildTxWithMessages BuildTxWithMessagesFn

	// PendingTxToProcessedMsgs is the function to convert pending tx to processed messages.
	PendingTxToProcessedMsgs PendingTxToProcessedMsgsFn

	// KeyringConfig is the keyring configuration.
	KeyringConfig KeyringConfig
}

func (bc *BroadcasterConfig) WithPendingTxToProcessedMsgsFn(fn PendingTxToProcessedMsgsFn) {
	bc.PendingTxToProcessedMsgs = fn
}

func (bc *BroadcasterConfig) WithBuildTxWithMessagesFn(fn BuildTxWithMessagesFn) {
	bc.BuildTxWithMessages = fn
}

func (bc BroadcasterConfig) Validate() error {
	if bc.ChainID == "" {
		return fmt.Errorf("chain id is empty")
	}

	_, err := sdk.ParseDecCoins(bc.GasPrice)
	if err != nil {
		return fmt.Errorf("failed to parse gas price: %s", bc.GasPrice)
	}

	if bc.Bech32Prefix == "" {
		return fmt.Errorf("bech32 prefix is empty")
	}

	if bc.BuildTxWithMessages == nil {
		return fmt.Errorf("build tx with messages is nil")
	}

	if bc.PendingTxToProcessedMsgs == nil {
		return fmt.Errorf("pending tx to processed msgs is nil")
	}

	if bc.GasAdjustment == 0 {
		return fmt.Errorf("gas adjustment is zero")
	}

	if bc.TxTimeout == 0 {
		return fmt.Errorf("tx timeout is zero")
	}

	return bc.KeyringConfig.Validate()
}

func (bc BroadcasterConfig) GetKeyringRecord(cdc codec.Codec, chainID string) (keyring.Keyring, *keyring.Record, error) {
	keyBase, err := keys.GetKeyBase(chainID, bc.KeyringConfig.HomePath, cdc, nil)
	if err != nil {
		return nil, nil, err
	} else if keyBase == nil {
		return nil, nil, fmt.Errorf("failed to get key base")
	}

	keyringRecord, err := bc.KeyringConfig.GetKeyRecord(keyBase, bc.Bech32Prefix)
	if err != nil {
		return nil, nil, err
	} else if keyringRecord == nil {
		return nil, nil, fmt.Errorf("keyring record is nil")
	}

	return keyBase, keyringRecord, nil
}

type KeyringConfig struct {
	// HomePath is the path to the keyring.
	HomePath string `json:"home_path"`

	// Name of key in keyring
	Name string `json:"name"`

	// Address of key in keyring
	Address string `json:"address"`

	// Mnemonic of key in keyring
	// Mnemonic string `json:"mnemonic"`
}

func (kc KeyringConfig) GetKeyRecord(keyBase keyring.Keyring, bech32Prefix string) (*keyring.Record, error) {
	if kc.Address != "" {
		addr, err := sdk.GetFromBech32(kc.Address, bech32Prefix)
		if err != nil {
			return nil, err
		}
		key, err := keyBase.KeyByAddress(sdk.AccAddress(addr))
		if err != nil {
			return nil, fmt.Errorf("failed to get key by address from keyring: %s", kc.Address)
		}

		return key, nil
	} else if kc.Name != "" {
		key, err := keyBase.Key(kc.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to get key from keyring: %s", kc.Name)
		}

		return key, nil
	}

	return nil, fmt.Errorf("keyring config is invalid")
}

func (kc KeyringConfig) Validate() error {
	if kc.Name == "" && kc.Address == "" {
		return fmt.Errorf("keyring config is invalid")
	}

	return nil
}
