package keys

import (
	"io"

	"path"

	"github.com/cosmos/go-bip39"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
)

func GetKeyDir(homePath string) string {
	return path.Join(homePath, ".keys")
}

func GetKeyBase(chainId string, dir string, cdc codec.Codec, userInput io.Reader) (keyring.Keyring, error) {
	return keyring.New(chainId, "os", GetKeyDir(dir), userInput, cdc)
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
