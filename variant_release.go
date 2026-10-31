//go:build !dev

package main

import m "github.com/plusev-terminal/go-plugin-common/v2/meta"

// Release builds: production ACL only, no name/version marker.
const (
	buildVariantName    = ""
	buildVariantVersion = ""
)

func networkTargets() []m.NetworkTargetRule {
	return productionNetworkTargets()
}
