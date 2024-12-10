package keys

import (
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// EncodeBech32AccAddr encodes the given account address to bech32 format
func EncodeBech32AccAddr(addr sdk.AccAddress, prefix string) (string, error) {
	return sdk.Bech32ifyAddressBytes(prefix, addr)
}

// DecodeBech32AccAddr decodes the given bech32 account address
func DecodeBech32AccAddr(addr string, prefix string) (sdk.AccAddress, error) {
	return sdk.GetFromBech32(addr, prefix)
}

var sdkConfigMutex sync.Mutex

// SetSDKContext sets the SDK config to the given bech32 prefixes
func SetSDKConfigContext(prefix string) func() {
	sdkConfigMutex.Lock()
	sdkConf := sdk.GetConfig()
	sdkConf.SetBech32PrefixForAccount(prefix, prefix+"pub")
	sdkConf.SetBech32PrefixForValidator(prefix+"valoper", prefix+"valoperpub")
	sdkConf.SetBech32PrefixForConsensusNode(prefix+"valcons", prefix+"valconspub")
	return sdkConfigMutex.Unlock
}
