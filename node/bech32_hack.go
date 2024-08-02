package node

import (
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

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
