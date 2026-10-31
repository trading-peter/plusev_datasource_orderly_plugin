package main

import (
	"reflect"
	"testing"
	"time"

	m "github.com/plusev-terminal/go-plugin-common/meta"
)

// wantRules pins the manifest rules table to Orderly's published budgets
// (retrieved 2026-10-31; per-page source URLs in network.go). Burst stays
// 0 (= full window, SDK default) everywhere: the docs publish per-window
// quotas and no smaller bucket.
var wantRules = []m.RateLimitRule{
	{ID: "public-info", Limit: 10, WindowMs: time.Second.Milliseconds(), Paths: []string{"/v1/public/info"}},
	{ID: "kline", Limit: 10, WindowMs: time.Second.Milliseconds(), Paths: []string{"/v1/kline"}},
	{ID: "kline-history", Limit: 5, WindowMs: (10 * time.Second).Milliseconds(), Paths: []string{"/v1/tv/kline_history"}},
	{ID: "orders", Limit: 10, WindowMs: time.Second.Milliseconds(), Scope: m.RateLimitScopeCredential, Paths: []string{"/v1/order", "/v1/client/order"}},
	{ID: "algo-orders", Limit: 1, WindowMs: time.Second.Milliseconds(), Scope: m.RateLimitScopeCredential, Paths: []string{"/v1/algo/order", "/v1/algo/client/order"}},
	{ID: "batch-order", Limit: 1, WindowMs: time.Second.Milliseconds(), Scope: m.RateLimitScopeCredential, Paths: []string{"/v1/batch-order", "/v1/client/batch-order"}},
	{ID: "positions", Limit: 30, WindowMs: (10 * time.Second).Milliseconds(), Scope: m.RateLimitScopeCredential, Paths: []string{"/v1/positions"}},
	{ID: "client-info", Limit: 10, WindowMs: time.Minute.Milliseconds(), Scope: m.RateLimitScopeCredential, Paths: []string{"/v1/client/info"}},
}

// TestManifestRateLimits pins the rate-limit declarations on the Orderly
// REST network targets so the numbers can't silently drift.
func TestManifestRateLimits(t *testing.T) {
	targets := productionNetworkTargets()

	byPattern := map[string]*m.NetworkTargetRule{}
	for i := range targets {
		byPattern[targets[i].Pattern] = &targets[i]
	}

	for _, pattern := range []string{
		"https://api.orderly.org/*",
		"https://testnet-api.orderly.org/*",
	} {
		t.Run(pattern, func(t *testing.T) {
			target := byPattern[pattern]
			if target == nil {
				t.Fatalf("missing network target %s", pattern)
			}
			if len(target.Limits) != len(wantRules) {
				t.Fatalf("expected %d limit rules, got %d", len(wantRules), len(target.Limits))
			}
			for i, want := range wantRules {
				got := target.Limits[i]
				if got.ID != want.ID || got.Limit != want.Limit || got.WindowMs != want.WindowMs {
					t.Errorf("rule %d drifted: got %+v, want id=%s limit=%d windowMs=%d", i, got, want.ID, want.Limit, want.WindowMs)
				}
				if got.Burst != 0 || got.PerKeyEnabled || got.CostWeighted {
					t.Errorf("rule %d shape drifted (want burst 0, not per-key, not weighted): %+v", i, got)
				}
				if got.Scope != want.Scope {
					t.Errorf("rule %d scope drifted: got %q, want %q", i, got.Scope, want.Scope)
				}
				if !reflect.DeepEqual(got.Paths, want.Paths) {
					t.Errorf("rule %d paths drifted: got %v, want %v", i, got.Paths, want.Paths)
				}
			}
			if target.TradingReserve == nil || *target.TradingReserve != 0.3 {
				t.Errorf("expected trading reserve 0.3, got %+v", target.TradingReserve)
			}
		})
	}

	for _, pattern := range []string{
		"wss://ws-evm.orderly.org/*",
		"wss://testnet-ws-evm.orderly.org/*",
	} {
		if target := byPattern[pattern]; target == nil || len(target.Limits) != 0 {
			t.Errorf("websocket target %s must stay unlimited", pattern)
		}
	}

	// Manifest limits replace the legacy per-command limiter: explicit
	// empty-slice opt-out, keeping the export alive.
	if limits := (&OrderlyPlugin{}).GetRateLimits(); len(limits) != 0 {
		t.Errorf("GetRateLimits must opt out with an empty slice, got %d entries", len(limits))
	}
}
