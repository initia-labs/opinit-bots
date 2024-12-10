package keys

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestEncodeBech32Addr(t *testing.T) {
	cases := []struct {
		title        string
		base         string
		bech32Prefix string
		expected     string
		err          bool
	}{
		{
			title:        "init",
			base:         "b8fb0b7c5158f5028ff0a8ce99712fb545cbf74e",
			bech32Prefix: "init",
			expected:     "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5",
			err:          false,
		},
		{
			title:        "abcdef",
			base:         "b8fb0b7c5158f5028ff0a8ce99712fb545cbf74e",
			bech32Prefix: "abcdef",
			expected:     "abcdef1hrasklz3tr6s9rls4r8fjuf0k4zuha6whsl7cn",
			err:          false,
		},
		{
			title:        "cosmos",
			base:         "b8fb0b7c5158f5028ff0a8ce99712fb545cbf74e",
			bech32Prefix: "cosmos",
			expected:     "cosmos1hrasklz3tr6s9rls4r8fjuf0k4zuha6wt4u7jk",
			err:          false,
		},
		{
			title:        "empty prefix",
			base:         "b8fb0b7c5158f5028ff0a8ce99712fb545cbf74e",
			bech32Prefix: "",
			expected:     "",
			err:          true,
		},
		{
			title:        "empty base",
			base:         "",
			bech32Prefix: "abcde",
			expected:     "",
			err:          false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.title, func(t *testing.T) {
			hexBase, err := hex.DecodeString(tc.base)
			require.NoError(t, err)

			addr, err := EncodeBech32AccAddr(hexBase, tc.bech32Prefix)
			if tc.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, addr)
			}
		})
	}
}

func TestDecodeBech32Addr(t *testing.T) {
	cases := []struct {
		title        string
		address      string
		bech32Prefix string
		expected     string
		err          bool
	}{
		{
			title:        "init",
			address:      "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5",
			bech32Prefix: "init",
			expected:     "b8fb0b7c5158f5028ff0a8ce99712fb545cbf74e",
			err:          false,
		},
		{
			title:        "abcdef",
			address:      "abcdef1hrasklz3tr6s9rls4r8fjuf0k4zuha6whsl7cn",
			bech32Prefix: "abcdef",
			expected:     "b8fb0b7c5158f5028ff0a8ce99712fb545cbf74e",
			err:          false,
		},
		{
			title:        "cosmos",
			address:      "cosmos1hrasklz3tr6s9rls4r8fjuf0k4zuha6wt4u7jk",
			bech32Prefix: "cosmos",
			expected:     "b8fb0b7c5158f5028ff0a8ce99712fb545cbf74e",
			err:          false,
		},
		{
			title:        "empty prefix",
			address:      "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5",
			bech32Prefix: "",
			expected:     "",
			err:          true,
		},
		{
			title:        "invalid prefix",
			address:      "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5",
			bech32Prefix: "init1",
			expected:     "",
			err:          true,
		},
		{
			title:        "invalid address",
			address:      "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9r1ude5",
			bech32Prefix: "init",
			expected:     "",
			err:          true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.title, func(t *testing.T) {
			base, err := DecodeBech32AccAddr(tc.address, tc.bech32Prefix)
			if tc.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				expected, err := hex.DecodeString(tc.expected)
				require.NoError(t, err)
				require.Equal(t, expected, base.Bytes())
			}
		})
	}
}

func TestSetSDKConfigContext(t *testing.T) {
	prefixes := []string{"init", "cosmos", "abcdef", "abcabcabc", ""}

	for _, prefix := range prefixes {
		t.Run(prefix, func(t *testing.T) {
			t.Parallel()
			unlock := SetSDKConfigContext(prefix)
			defer unlock()

			// test the sdk config
			conf := sdk.GetConfig()
			require.Equal(t, prefix, conf.GetBech32AccountAddrPrefix())
			require.Equal(t, prefix+"pub", conf.GetBech32AccountPubPrefix())
			require.Equal(t, prefix+"valoper", conf.GetBech32ValidatorAddrPrefix())
			require.Equal(t, prefix+"valoperpub", conf.GetBech32ValidatorPubPrefix())
		})
	}
}
