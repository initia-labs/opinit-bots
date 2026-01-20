package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultConfigIncludesOracleRelay(t *testing.T) {
	cfg := DefaultConfig()

	bz, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(t, err)

	jsonStr := string(bz)
	require.Contains(t, jsonStr, "oracle_relay", "default config JSON should include oracle_relay section")
	require.Contains(t, jsonStr, `"enable"`, "oracle_relay should have enable field")
	require.Contains(t, jsonStr, `"interval"`, "oracle_relay should have interval field")
	require.Contains(t, jsonStr, `"currency_pairs"`, "oracle_relay should have currency_pairs field")

	var parsed Config
	err = json.Unmarshal(bz, &parsed)
	require.NoError(t, err)

	require.False(t, parsed.OracleRelay.Enable, "oracle relay should be disabled by default")
	require.Equal(t, int64(30), parsed.OracleRelay.Interval, "default interval should be 30")
	require.NotNil(t, parsed.OracleRelay.CurrencyPairs, "currency pairs should not be nil")
	require.Empty(t, parsed.OracleRelay.CurrencyPairs, "currency pairs should be empty by default")
}

func TestConfigWithoutOracleRelayFails(t *testing.T) {
	cfg := DefaultConfig()
	cfg.OracleRelay.Interval = 0

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "oracle relay")
}
