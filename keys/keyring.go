package keys

import (
	"io"

	"path"

	"github.com/cosmos/go-bip39"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
)

// GetKeyDir returns the key directory based on the home path and chain ID.
func GetKeyDir(homePath string, chainId string) string {
	return path.Join(homePath, chainId)
}

// GetKeyBase returns a keybase based on the given chain ID and directory.
// If the directory is empty, an in-memory keybase is returned.
func GetKeyBase(chainId string, dir string, cdc codec.Codec, userInput io.Reader) (keyring.Keyring, error) {
	if dir == "" {
		return keyring.NewInMemory(cdc, Option()), nil
	}
	return keyring.New(chainId, "test", GetKeyDir(dir, chainId), userInput, cdc, Option())
}

// CreateMnemonic generates a new mnemonic.
func CreateMnemonic() (string, error) {
	entropySeed, err := bip39.NewEntropy(256)
	if err != nil {
		return "", err
	}
	mnemonic, err := bip39.NewMnemonic(entropySeed)
	if err != nil {
		return "", err
	}
	return mnemonic, nil
}
