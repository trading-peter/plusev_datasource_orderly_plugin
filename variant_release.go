//go:build !dev

package main

import m "github.com/plusev-terminal/go-plugin-common/meta"

// Release builds: production ACL only, no name/version marker.
const (
	buildVariantName    = ""
	buildVariantVersion = ""
)

func networkTargets() []m.NetworkTargetRule {
	return productionNetworkTargets()
}
