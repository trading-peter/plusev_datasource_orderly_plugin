//go:build wasip1

package orderly

import (
	"testing"

	"github.com/trading-peter/plusev_datasource_orderly_plugin/urls"
)

// The client carries the resolved URLs it was built with; every REST
// call prefixes c.baseURL and the stream dials {wsBaseURL}/ws/stream/
// {accountID} (ORD-701).
func TestNewClientStoresResolvedURLs(t *testing.T) {
	tests := []struct {
		name      string
		baseURL   string
		wsURL     string
		isTestnet bool
	}{
		{"production defaults", urls.DefaultBaseURL, urls.DefaultWSBaseURL, false},
		{"testnet hosts", urls.TestnetBaseURL, urls.TestnetWSBaseURL, true},
		{"sandbox override", "http://127.0.0.1:8080/sandbox/k1/orderly/", "ws://127.0.0.1:8080/sandbox/k1/orderly/", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient(nil, tt.baseURL, tt.wsURL, tt.isTestnet)

			if c.baseURL != tt.baseURL {
				t.Errorf("baseURL = %q, want %q", c.baseURL, tt.baseURL)
			}

			if c.wsBaseURL != tt.wsURL {
				t.Errorf("wsBaseURL = %q, want %q", c.wsBaseURL, tt.wsURL)
			}

			if c.isTestnet != tt.isTestnet {
				t.Errorf("isTestnet = %v, want %v", c.isTestnet, tt.isTestnet)
			}
		})
	}
}

// The URL override fields are declared so connections can point at the
// sandbox dialect (ORD-701); the credentials stay encrypted.
func TestGetConfigFieldsDeclaresURLOverrides(t *testing.T) {
	fields := NewClient(nil, urls.DefaultBaseURL, urls.DefaultWSBaseURL, false).GetConfigFields()

	byName := make(map[string]int, len(fields))
	for i, f := range fields {
		byName[f.Name] = i
	}

	for _, name := range []string{"accountID", "apiKey", "secretKey", "testnet", "baseUrl", "wsUrl"} {
		if _, ok := byName[name]; !ok {
			t.Errorf("GetConfigFields is missing field %q", name)
		}
	}

	base := fields[byName["baseUrl"]]
	if base.Encrypt {
		t.Error("baseUrl must be a plain field")
	}

	if base.Placeholder == "" {
		t.Error("baseUrl should document the default via its placeholder")
	}

	ws := fields[byName["wsUrl"]]
	if ws.Encrypt {
		t.Error("wsUrl must be a plain field")
	}

	testnet := fields[byName["testnet"]]
	if testnet.Type != "boolean" {
		t.Errorf("testnet type = %q, want boolean", testnet.Type)
	}
}
