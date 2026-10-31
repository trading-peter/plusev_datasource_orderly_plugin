package main

import (
	"time"

	m "github.com/plusev-terminal/go-plugin-common/v2/meta"
)

// Manifest rate limits for the Orderly REST hosts (api.orderly.org and
// testnet-api.orderly.org; the docs list both as servers of the same API).
//
// Sources (retrieved 2026-10-31), all under
// https://orderly.network/docs/build-on-omnichain/:
//   - introduction: "Rate limits are counted per Orderly Key" → private
//     endpoints use PerCredential buckets; public market-data endpoints
//     state "per IP address" → per-host shared bucket (default scope).
//   - restful-api/public/get-available-symbols        GET /v1/public/info       10 req / 1 s per IP
//   - restful-api/public/get-kline                   GET /v1/kline             10 req / 1 s (public → per IP)
//   - restful-api/public/get-kline-history           GET /v1/tv/kline_history   5 req / 10 s per IP
//   - restful-api/private/get-all-positions-info     GET /v1/positions         30 req / 10 s per user
//   - restful-api/private/get-account-information    GET /v1/client/info       10 req / 60 s per key
//   - restful-api/private/create-order               POST /v1/order            10 req / 1 s per key
//   - restful-api/private/cancel-order               DELETE /v1/order          10 req / 1 s per key
//   - restful-api/private/get-orders                 GET /v1/orders            10 req / 1 s per key
//   - restful-api/private/cancel-all-pending-orders  DELETE /v1/orders         10 req / 1 s per key
//   - restful-api/private/cancel-order-by-client_order_id  DELETE /v1/client/order  10 req / 1 s per key
//   - restful-api/private/create-algo-order          POST /v1/algo/order        1 req / 1 s per key
//   - restful-api/private/get-algo-orders            GET /v1/algo/orders        5 req / 1 s per key
//   - restful-api/private/cancel-algo-order          DELETE /v1/algo/order      5 req / 1 s per key
//   - restful-api/private/cancel-all-pending-algo-orders  DELETE /v1/algo/orders  5 req / 1 s per key
//   - restful-api/private/cancel-algo-order-by-client_order_id  DELETE /v1/algo/client/order  5 req / 1 s per key
//   - restful-api/private/batch-create-order         POST /v1/batch-order       1 req / 1 s per key
//   - restful-api/private/batch-cancel-orders        DELETE /v1/batch-order    10 req / 1 s per key
//
// Rules match URL path PREFIXES and cannot distinguish HTTP methods, so
// where one path prefix carries different per-method budgets the strictest
// documented rate is declared (a safe floor; users can raise it in
// Terminal's Rate Limits UI):
//   - /v1/algo/order also prefixes /v1/algo/orders (GET/DELETE = 5 req/s),
//     but POST /v1/algo/order = 1 req/s → the whole prefix is declared 1 rps.
//   - /v1/batch-order is POST = 1 req/s vs DELETE = 10 req/s → 1 rps.
//
// Not covered: GET /v1/client/holding (balances) has no published limit
// page in the current docs, so it keeps adaptive 429/Retry-After handling
// only. The WebSocket hosts carry no HTTP request rules (the host, not the
// plugin requester, owns the WS connections).

// orderlyTradingReserve reserves part of every rule's budget for Trading
// class requests: order create/cancel share both REST hosts with data
// pulls (markets, OHLCV, positions), so keep 30% headroom for orders
// (per-key budgets are small — 10 req/s on the order paths).
var orderlyTradingReserve = 0.3

// productionNetworkTargets are the hosts every build may talk to: the
// Orderly production and testnet REST/WS pairs. The REST hosts carry the
// manifest rate limits; the WS hosts don't (no HTTP budget published).
func productionNetworkTargets() []m.NetworkTargetRule {
	return []m.NetworkTargetRule{
		{
			Pattern:        "https://api.orderly.org/*",
			Limits:         orderlyRestLimits(),
			TradingReserve: &orderlyTradingReserve,
		},
		{
			Pattern:        "https://testnet-api.orderly.org/*",
			Limits:         orderlyRestLimits(),
			TradingReserve: &orderlyTradingReserve,
		},
		{Pattern: "wss://ws-evm.orderly.org/*"},
		{Pattern: "wss://testnet-ws-evm.orderly.org/*"},
	}
}

// orderlyRestLimits declares Orderly's published per-endpoint budgets for
// one REST host. Private endpoints are keyed per Orderly Key
// (PerCredential); public market-data endpoints are per IP (default
// per-host scope). Bursts are left at the window's limit (bucket = one
// full window), matching the docs' per-window quotas.
func orderlyRestLimits() []m.RateLimitRule {
	return []m.RateLimitRule{
		// Public market data — budgets per IP address.
		m.NewRateLimitRule("public-info", 10, time.Second).
			WithLabel("Public symbol info (per IP)").
			ForPaths("/v1/public/info"), // 10 req/1s per IP: restful-api/public/get-available-symbols
		m.NewRateLimitRule("kline", 10, time.Second).
			WithLabel("Recent klines (per IP)").
			ForPaths("/v1/kline"), // 10 req/1s: restful-api/public/get-kline
		m.NewRateLimitRule("kline-history", 5, 10*time.Second).
			WithLabel("Kline history (per IP)").
			ForPaths("/v1/tv/kline_history"), // 5 req/10s per IP: restful-api/public/get-kline-history

		// Private endpoints — budgets per Orderly Key.
		m.NewRateLimitRule("orders", 10, time.Second).
			WithLabel("Orders (per key)").
			PerCredential().
			// 10 req/1s: create-order, cancel-order, get-orders,
			// cancel-all-pending-orders, cancel-order-by-client_order_id.
			// "/v1/order" also prefixes "/v1/orders" (same 10 req/s).
			ForPaths("/v1/order", "/v1/client/order"),
		m.NewRateLimitRule("algo-orders", 1, time.Second).
			WithLabel("Algo orders (per key)").
			PerCredential().
			// POST /v1/algo/order = 1 req/1s (create-algo-order); reads and
			// cancels on the same prefix are 5 req/1s but rules cannot split
			// by method, so the strictest rate covers the whole prefix.
			ForPaths("/v1/algo/order", "/v1/algo/client/order"),
		m.NewRateLimitRule("batch-order", 1, time.Second).
			WithLabel("Batch orders (per key)").
			PerCredential().
			// POST /v1/batch-order = 1 req/1s (batch-create-order); the
			// DELETE batch-cancel on the same path is 10 req/1s — strictest
			// wins. /v1/client/batch-order (batch cancel by client id) is
			// grouped here at the same conservative rate.
			ForPaths("/v1/batch-order", "/v1/client/batch-order"),
		m.NewRateLimitRule("positions", 30, 10*time.Second).
			WithLabel("Positions (per key)").
			PerCredential().
			ForPaths("/v1/positions"), // 30 req/10s per user: get-all-positions-info
		m.NewRateLimitRule("client-info", 10, time.Minute).
			WithLabel("Client info (per key)").
			PerCredential().
			ForPaths("/v1/client/info"), // 10 req/60s: get-account-information
	}
}
