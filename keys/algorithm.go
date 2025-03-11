package keys

import (
	"github.com/cosmos/go-bip39"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// EthSecp256k1Type defines the ECDSA secp256k1 used on Ethereum
	EthSecp256k1Type = hd.PubKeyType(KeyType)
)

var (
	_ keyring.SignatureAlgo = EthSecp256k1

	// EthSecp256k1 uses the Bitcoin secp256k1 ECDSA parameters.
	EthSecp256k1 = ethSecp256k1Algo{}
)

const DefaultFullBIP44Path = "m/44'/60'/0'/0/0"

type ethSecp256k1Algo struct{}

// Name returns eth_secp256k1
func (s ethSecp256k1Algo) Name() hd.PubKeyType {
	return EthSecp256k1Type
}

// Derive derives and returns the secp256k1 private key for the given seed and HD path.
func (s ethSecp256k1Algo) Derive() hd.DeriveFn {
	return func(mnemonic, bip39Passphrase, hdPath string) ([]byte, error) {
		// override the default BIP44 path to match Ethereum derivation
		if hdPath == sdk.GetConfig().GetFullBIP44Path() {
			hdPath = DefaultFullBIP44Path
		}

		seed, err := bip39.NewSeedWithErrorChecking(mnemonic, bip39Passphrase)
		if err != nil {
			return nil, err
		}

		masterPriv, ch := hd.ComputeMastersFromSeed(seed)
		if len(hdPath) == 0 {
			return masterPriv[:], nil
		}
		derivedKey, err := hd.DerivePrivateKeyForPath(masterPriv, ch, hdPath)

		return derivedKey, err
	}
}

// Generate generates a secp256k1 private key from the given bytes.
func (s ethSecp256k1Algo) Generate() hd.GenerateFn {
	return func(bz []byte) types.PrivKey {
		bzArr := make([]byte, PrivKeySize)
		copy(bzArr, bz)

		return &PrivKey{Key: bzArr}
	}
}
