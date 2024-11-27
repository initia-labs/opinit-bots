package keys

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/stretchr/testify/require"
)

func TestGetKeyDir(t *testing.T) {
	require.Equal(t, "homePath/chainId", GetKeyDir("homePath", "chainId"))
	require.Equal(t, "chainId", GetKeyDir("", "chainId"))
}

func TestGetKeyBase(t *testing.T) {
	keybase, err := GetKeyBase("chainId", "dir", nil, nil)
	require.NoError(t, err)
	require.NotNil(t, keybase)
	require.Equal(t, keyring.BackendTest, keybase.Backend())

	keybase, err = GetKeyBase("chainId", "", nil, nil)
	require.NoError(t, err)
	require.NotNil(t, keybase)
	require.Equal(t, keyring.BackendMemory, keybase.Backend())
}
