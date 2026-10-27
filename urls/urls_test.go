package urls

import "testing"

// ORD-701 resolution matrix: an explicit config value wins, then the
// testnet flag, then the production host.
func TestResolveURLs(t *testing.T) {
	tests := []struct {
		name      string
		baseURL   string
		wsURL     string
		isTestnet bool
		wantBase  string
		wantWS    string
	}{
		{
			name:     "empty config, mainnet flag off — production hosts",
			wantBase: DefaultBaseURL,
			wantWS:   DefaultWSBaseURL,
		},
		{
			name:      "empty config, testnet on — testnet hosts",
			isTestnet: true,
			wantBase:  TestnetBaseURL,
			wantWS:    TestnetWSBaseURL,
		},
		{
			name:     "explicit base only — explicit base, production ws",
			baseURL:  "http://127.0.0.1:8080/sandbox/k1/orderly/",
			wantBase: "http://127.0.0.1:8080/sandbox/k1/orderly/",
			wantWS:   DefaultWSBaseURL,
		},
		{
			name:     "explicit ws only — production base, explicit ws",
			wsURL:    "ws://127.0.0.1:8080/sandbox/k1/orderly/",
			wantBase: DefaultBaseURL,
			wantWS:   "ws://127.0.0.1:8080/sandbox/k1/orderly/",
		},
		{
			name:      "explicit values beat the testnet flag",
			baseURL:   "http://127.0.0.1:8080/sandbox/k1/orderly/",
			wsURL:     "ws://127.0.0.1:8080/sandbox/k1/orderly/",
			isTestnet: true,
			wantBase:  "http://127.0.0.1:8080/sandbox/k1/orderly/",
			wantWS:    "ws://127.0.0.1:8080/sandbox/k1/orderly/",
		},
		{
			name:      "testnet base with explicit ws override",
			wsURL:     "ws://127.0.0.1:8080/sandbox/k1/orderly/",
			isTestnet: true,
			wantBase:  TestnetBaseURL,
			wantWS:    "ws://127.0.0.1:8080/sandbox/k1/orderly/",
		},
		{
			name:     "whitespace-only values count as empty",
			baseURL:  "   ",
			wsURL:    "\t",
			wantBase: DefaultBaseURL,
			wantWS:   DefaultWSBaseURL,
		},
		{
			name:     "explicit values are trimmed",
			baseURL:  " https://api.example.com ",
			wsURL:    " wss://ws.example.com ",
			wantBase: "https://api.example.com",
			wantWS:   "wss://ws.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBase, gotWS := ResolveURLs(tt.baseURL, tt.wsURL, tt.isTestnet)

			if gotBase != tt.wantBase {
				t.Errorf("baseURL = %q, want %q", gotBase, tt.wantBase)
			}

			if gotWS != tt.wantWS {
				t.Errorf("wsURL = %q, want %q", gotWS, tt.wantWS)
			}
		})
	}
}
