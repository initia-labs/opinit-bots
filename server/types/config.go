package types

import "errors"

type ServerConfig struct {
	Address      string `json:"address"`
	AllowOrigins string `json:"allow_origins"`
	AllowHeaders string `json:"allow_headers"`
	AllowMethods string `json:"allow_methods"`
}

func (s ServerConfig) Validate() error {
	if s.Address == "" {
		return errors.New("address is required")
	}
	return nil
}
