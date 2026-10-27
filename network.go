package main

import m "github.com/plusev-terminal/go-plugin-common/meta"

// productionNetworkTargets are the hosts every build may talk to: the
// Orderly production and testnet REST/WS pairs.
func productionNetworkTargets() []m.NetworkTargetRule {
	return []m.NetworkTargetRule{
		{Pattern: "https://api.orderly.org/*"},
		{Pattern: "https://testnet-api.orderly.org/*"},
		{Pattern: "wss://ws-evm.orderly.org/*"},
		{Pattern: "wss://testnet-ws-evm.orderly.org/*"},
	}
}
