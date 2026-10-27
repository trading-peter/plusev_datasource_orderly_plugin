// Package urls resolves the Orderly plugin's REST and WebSocket base
// URLs from its config (ORD-701): an explicit field wins, then the
// testnet flag, then the production host. Kept free of go-plugin-common
// imports so the resolution matrix is native `go test` friendly.
package urls

import "strings"

// Orderly network hosts. The testnet toggle picks the testnet pair;
// explicit baseUrl/wsUrl config fields win over both (ORD-701).
const (
	DefaultBaseURL   = "https://api.orderly.org"
	DefaultWSBaseURL = "wss://ws-evm.orderly.org"

	TestnetBaseURL   = "https://testnet-api.orderly.org"
	TestnetWSBaseURL = "wss://testnet-ws-evm.orderly.org"
)

// ResolveURLs picks the REST and WebSocket base URLs: an explicit config
// value wins; otherwise the testnet flag selects the testnet host;
// otherwise the production default.
func ResolveURLs(baseURL, wsURL string, isTestnet bool) (string, string) {
	resolvedBase := DefaultBaseURL
	resolvedWS := DefaultWSBaseURL

	if isTestnet {
		resolvedBase = TestnetBaseURL
		resolvedWS = TestnetWSBaseURL
	}

	if baseURL = strings.TrimSpace(baseURL); baseURL != "" {
		resolvedBase = baseURL
	}

	if wsURL = strings.TrimSpace(wsURL); wsURL != "" {
		resolvedWS = wsURL
	}

	return resolvedBase, resolvedWS
}
