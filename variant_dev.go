//go:build dev

package main

import m "github.com/plusev-terminal/go-plugin-common/meta"

// Dev builds (go build -tags dev): marked in the plugin list and allowed
// to reach the Exchange Sandbox dialect on the loopback (ORD-701).
const (
	buildVariantName    = " (dev)"
	buildVariantVersion = "-dev"
)

// The host's network ACL matches the scheme and host exactly and only
// wildcards the path (pluginmgr/types/networkTargetRules.go — url.Parse
// rejects "* ports"), so the loopback endpoints are enumerated. Port 8080
// is Terminal's dev default; add a line here for other ports.
func networkTargets() []m.NetworkTargetRule {
	return append(productionNetworkTargets(),
		m.NetworkTargetRule{Pattern: "http://127.0.0.1:8080/*"},
		m.NetworkTargetRule{Pattern: "ws://127.0.0.1:8080/*"},
		m.NetworkTargetRule{Pattern: "http://localhost:8080/*"},
		m.NetworkTargetRule{Pattern: "ws://localhost:8080/*"},
	)
}
