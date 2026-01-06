package types

import "errors"

type ServerConfig struct {
	Address               string `json:"address"`
	AllowOrigins          string `json:"allow_origins"`
	AllowHeaders          string `json:"allow_headers"`
	AllowMethods          string `json:"allow_methods"`
	MetricsUpdateInterval int64  `json:"metrics_update_interval"`
}

func (s ServerConfig) Validate() error {
	if s.Address == "" {
		return errors.New("address is required")
	}
	if s.MetricsUpdateInterval < 0 {
		return errors.New("metrics update interval must be non-negative")
	}
	return nil
}
