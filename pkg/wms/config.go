//nolint:gosec
package wms

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Config struct {
	Endpoints       []Endpoint
	Timeout         time.Duration
	OverrideOffices []OverrideOffice
}

type Auth struct {
	Username string
	Password string
}

type OverrideOffice struct {
	Country   string
	OfficeIDs []int32
}

type ConfigFileSchema struct {
	Endpoints map[string]struct {
		Enabled  bool   `json:"enabled"`
		Username string `json:"username"`
		Password string `json:"password"`
		URI      string `json:"uri"`
	} `json:"endpoints"`
	Timeout         int                `json:"timeout"`
	OverrideOffices map[string][]int32 `json:"override_offices"`
}

func (s ConfigFileSchema) Config() Config {
	endpoints := make([]Endpoint, 0, len(s.Endpoints))

	for country, endpoint := range s.Endpoints {
		endpoints = append(endpoints, Endpoint{
			Country: country,
			Enabled: endpoint.Enabled,
			Auth: Auth{
				Username: endpoint.Username,
				Password: endpoint.Password,
			},
			URI: endpoint.URI,
		})
	}

	overrideOffices := make([]OverrideOffice, 0, len(s.OverrideOffices))

	for country, officeIDs := range s.OverrideOffices {
		overrideOffices = append(overrideOffices, OverrideOffice{
			Country:   country,
			OfficeIDs: officeIDs,
		})
	}

	return Config{
		Endpoints:       endpoints,
		Timeout:         time.Second * time.Duration(s.Timeout),
		OverrideOffices: overrideOffices,
	}
}

func ReadConfigFile(name string) (Config, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return Config{}, fmt.Errorf("wms: read config file: %w", err)
	}

	return ConfigFromBytes(data)
}

func ConfigFromBytes(data []byte) (Config, error) {
	var schema ConfigFileSchema

	if err := json.Unmarshal(data, &schema); err != nil {
		return Config{}, fmt.Errorf("wms: unmarshal config: %w", err)
	}

	return schema.Config(), nil
}
