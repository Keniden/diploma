package wms

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

const sampleConfig = `{
	"endpoints": {
		"russia": {
			"enabled": true,
			"username": "u",
			"password": "p",
			"uri": "http://localhost:8081"
		},
		"belarus": {
			"enabled": false,
			"username": "bu",
			"password": "bp",
			"uri": "http://localhost:8082"
		}
	},
	"timeout": 15,
	"override_offices": {
		"belarus": [42, 43]
	}
}`

func TestConfigFromBytes(t *testing.T) {
	config, err := ConfigFromBytes([]byte(sampleConfig))
	if err != nil {
		t.Fatalf("ConfigFromBytes failed: %v", err)
	}

	if config.Timeout != 15*time.Second {
		t.Fatalf("timeout mismatch: %v", config.Timeout)
	}

	if len(config.Endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(config.Endpoints))
	}

	endpoints := map[string]Endpoint{}
	for _, e := range config.Endpoints {
		endpoints[e.Country] = e
	}

	russia, ok := endpoints["russia"]
	if !ok || !russia.Enabled || russia.URI != "http://localhost:8081" ||
		russia.Auth.Username != "u" || russia.Auth.Password != "p" {
		t.Fatalf("russia endpoint mismatch: %+v", russia)
	}

	if !ok || endpoints["belarus"].Enabled {
		t.Fatalf("belarus endpoint must be disabled: %+v", endpoints["belarus"])
	}

	if len(config.OverrideOffices) != 1 ||
		config.OverrideOffices[0].Country != "belarus" ||
		len(config.OverrideOffices[0].OfficeIDs) != 2 {
		t.Fatalf("override offices mismatch: %+v", config.OverrideOffices)
	}
}

func TestConfigFromBytesInvalidJSON(t *testing.T) {
	if _, err := ConfigFromBytes([]byte(`{invalid`)); err == nil {
		t.Fatal("invalid JSON must fail")
	}
}

func TestReadConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wms.json")

	if err := os.WriteFile(path, []byte(sampleConfig), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	config, err := ReadConfigFile(path)
	if err != nil {
		t.Fatalf("ReadConfigFile failed: %v", err)
	}

	if len(config.Endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(config.Endpoints))
	}
}

func TestReadConfigFileMissing(t *testing.T) {
	if _, err := ReadConfigFile(filepath.Join(t.TempDir(), "nope.json")); err == nil {
		t.Fatal("missing file must fail")
	}
}
