package types

import (
	"github.com/pkg/errors"
)

// OracleRelayConfig holds the configuration for the oracle relay handler
type OracleRelayConfig struct {
	// Enable enables the oracle relay feature
	Enable bool `json:"enable"`

	// Interval is the time interval between relay attempts in seconds
	Interval int64 `json:"interval"`
}

// DefaultOracleRelayConfig returns the default oracle relay configuration
func DefaultOracleRelayConfig() OracleRelayConfig {
	return OracleRelayConfig{
		Enable:   false,
		Interval: 3,
	}
}

// Validate validates the oracle relay configuration
func (cfg OracleRelayConfig) Validate() error {
	if cfg.Interval <= 0 {
		return errors.New("oracle relay interval must be greater than zero")
	}
	return nil
}
