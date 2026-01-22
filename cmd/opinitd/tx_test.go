package main

import (
	"testing"
	"time"

	"cosmossdk.io/x/feegrant"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/stretchr/testify/require"

	"github.com/initia-labs/opinit-bots/types"
)

func TestHasAuthzGrant(t *testing.T) {
	cases := []struct {
		name       string
		grants     []*authz.Grant
		msgTypeUrl string
		expected   bool
	}{
		{
			name:       "empty grants list",
			grants:     []*authz.Grant{},
			msgTypeUrl: types.MsgUpdateOracleTypeUrl,
			expected:   false,
		},
		{
			name: "grant exists for message type",
			grants: []*authz.Grant{
				createAuthzGrant(t, types.MsgUpdateOracleTypeUrl),
			},
			msgTypeUrl: types.MsgUpdateOracleTypeUrl,
			expected:   true,
		},
		{
			name: "grant does not exist for message type",
			grants: []*authz.Grant{
				createAuthzGrant(t, types.MsgRelayOracleTypeUrl),
			},
			msgTypeUrl: types.MsgUpdateOracleTypeUrl,
			expected:   false,
		},
		{
			name: "multiple grants, target exists",
			grants: []*authz.Grant{
				createAuthzGrant(t, types.MsgUpdateOracleTypeUrl),
				createAuthzGrant(t, types.MsgRelayOracleTypeUrl),
			},
			msgTypeUrl: types.MsgRelayOracleTypeUrl,
			expected:   true,
		},
		{
			name: "multiple grants, target does not exist",
			grants: []*authz.Grant{
				createAuthzGrant(t, types.MsgUpdateOracleTypeUrl),
				createAuthzGrant(t, types.MsgRelayOracleTypeUrl),
			},
			msgTypeUrl: types.MsgAuthzExecTypeUrl,
			expected:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := hasAuthzGrant(tc.grants, tc.msgTypeUrl)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestHasAllMsgTypes(t *testing.T) {
	requiredTypes := []string{
		types.MsgRelayOracleTypeUrl,
		types.MsgUpdateOracleTypeUrl,
		types.MsgAuthzExecTypeUrl,
	}

	cases := []struct {
		name          string
		grant         *feegrant.Grant
		requiredTypes []string
		expectResult  bool
		expectError   bool
	}{
		{
			name:          "nil grant",
			grant:         nil,
			requiredTypes: requiredTypes,
			expectResult:  false,
			expectError:   false,
		},
		{
			name:          "grant with all required types",
			grant:         createFeegrant(t, requiredTypes),
			requiredTypes: requiredTypes,
			expectResult:  true,
			expectError:   false,
		},
		{
			name: "grant missing one type",
			grant: createFeegrant(t, []string{
				types.MsgRelayOracleTypeUrl,
				types.MsgUpdateOracleTypeUrl,
			}),
			requiredTypes: requiredTypes,
			expectResult:  false,
			expectError:   false,
		},
		{
			name:          "grant with empty allowed messages (allows all)",
			grant:         createFeegrant(t, []string{}),
			requiredTypes: requiredTypes,
			expectResult:  true,
			expectError:   false,
		},
		{
			name: "grant with extra types",
			grant: createFeegrant(t, []string{
				types.MsgRelayOracleTypeUrl,
				types.MsgUpdateOracleTypeUrl,
				types.MsgAuthzExecTypeUrl,
				"/extra.v1.MsgExtra",
			}),
			requiredTypes: requiredTypes,
			expectResult:  true,
			expectError:   false,
		},
		{
			name: "grant with only some required types",
			grant: createFeegrant(t, []string{
				types.MsgRelayOracleTypeUrl,
			}),
			requiredTypes: requiredTypes,
			expectResult:  false,
			expectError:   false,
		},
		{
			name:          "grant with basic allowance (no msg restriction)",
			grant:         createBasicFeegrant(t),
			requiredTypes: requiredTypes,
			expectResult:  false,
			expectError:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := hasAllMsgTypes(tc.grant, tc.requiredTypes)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectResult, result)
			}
		})
	}
}

func createAuthzGrant(t *testing.T, msgTypeUrl string) *authz.Grant {
	auth := authz.NewGenericAuthorization(msgTypeUrl)

	registry := codectypes.NewInterfaceRegistry()
	authz.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	anyAuth, err := codectypes.NewAnyWithValue(auth)
	require.NoError(t, err)

	var authI authz.Authorization
	err = cdc.UnpackAny(anyAuth, &authI)
	require.NoError(t, err)

	return &authz.Grant{
		Authorization: anyAuth,
		Expiration:    nil,
	}
}

func createFeegrant(t *testing.T, allowedMsgs []string) *feegrant.Grant {
	basicAllowance := &feegrant.BasicAllowance{}
	allowedMsgAllowance, err := feegrant.NewAllowedMsgAllowance(basicAllowance, allowedMsgs)
	require.NoError(t, err)

	registry := codectypes.NewInterfaceRegistry()
	feegrant.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	anyAllowance, err := codectypes.NewAnyWithValue(allowedMsgAllowance)
	require.NoError(t, err)

	var allowanceI feegrant.FeeAllowanceI
	err = cdc.UnpackAny(anyAllowance, &allowanceI)
	require.NoError(t, err)

	return &feegrant.Grant{
		Granter:   "granter_address",
		Grantee:   "grantee_address",
		Allowance: anyAllowance,
	}
}

func createBasicFeegrant(t *testing.T) *feegrant.Grant {
	basicAllowance := &feegrant.BasicAllowance{
		SpendLimit: nil,
		Expiration: nil,
	}

	registry := codectypes.NewInterfaceRegistry()
	feegrant.RegisterInterfaces(registry)

	anyAllowance, err := codectypes.NewAnyWithValue(basicAllowance)
	require.NoError(t, err)

	expiration := time.Now().Add(24 * time.Hour)
	basicAllowance.Expiration = &expiration

	return &feegrant.Grant{
		Granter:   "granter_address",
		Grantee:   "grantee_address",
		Allowance: anyAllowance,
	}
}
