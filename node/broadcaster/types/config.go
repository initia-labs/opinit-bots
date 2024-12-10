package types

import (
	"context"
	"fmt"
	"time"

	"github.com/initia-labs/opinit-bots/keys"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type BuildTxWithMsgsFn func(context.Context, []sdk.Msg) ([]byte, string, error)
type MsgsFromTxFn func([]byte) ([]sdk.Msg, error)

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

	if bc.GasAdjustment == 0 {
		return fmt.Errorf("gas adjustment is zero")
	}

	if bc.TxTimeout == 0 {
		return fmt.Errorf("tx timeout is zero")
	}

	return nil
}

func (bc BroadcasterConfig) GetKeyringRecord(cdc codec.Codec, keyringConfig *KeyringConfig, homePath string) (keyring.Keyring, *keyring.Record, error) {
	if keyringConfig == nil {
		return nil, nil, fmt.Errorf("keyring config cannot be nil")
	}

	keyBase, err := keys.GetKeyBase(bc.ChainID, homePath, cdc, nil)
	if err != nil {
		return nil, nil, err
	} else if keyBase == nil {
		return nil, nil, fmt.Errorf("failed to get key base")
	}

	keyringRecord, err := keyringConfig.GetKeyRecord(keyBase, bc.Bech32Prefix)
	if err != nil {
		return nil, nil, err
	} else if keyringRecord == nil {
		return nil, nil, fmt.Errorf("keyring record is nil")
	}

	return keyBase, keyringRecord, nil
}

type KeyringConfig struct {
	// Name of key in keyring
	Name string `json:"name"`

	// Address of key in keyring
	Address string `json:"address"`

	// FeeGranter is the fee granter.
	FeeGranter *KeyringConfig

	// BuildTxWithMsgs is the function to build a transaction with messages.
	BuildTxWithMsgs BuildTxWithMsgsFn

	// MsgsFromTx is the function to convert pending tx to processed messages.
	MsgsFromTx MsgsFromTxFn
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

func (kc *KeyringConfig) WithPendingTxToProcessedMsgsFn(fn MsgsFromTxFn) {
	kc.MsgsFromTx = fn
}

func (kc *KeyringConfig) WithBuildTxWithMessagesFn(fn BuildTxWithMsgsFn) {
	kc.BuildTxWithMsgs = fn
}

func (kc KeyringConfig) Validate() error {
	if kc.Name == "" && kc.Address == "" {
		return fmt.Errorf("keyring config is invalid")
	}
	return nil
}
