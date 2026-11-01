package main

import (
	"fmt"

	ex "github.com/plusev-terminal/go-plugin-common/v2/datasrc/exchange"
	acct "github.com/plusev-terminal/go-plugin-common/v2/datasrc/exchange/account"
	"github.com/plusev-terminal/go-plugin-common/v2/logging"
	m "github.com/plusev-terminal/go-plugin-common/v2/meta"
	"github.com/plusev-terminal/go-plugin-common/v2/plugin"
	"github.com/plusev-terminal/go-plugin-common/v2/requester"
	tt "github.com/plusev-terminal/go-plugin-common/v2/trading"
	"github.com/trading-peter/plusev_datasource_orderly_plugin/orderly"
	"github.com/trading-peter/plusev_datasource_orderly_plugin/urls"
)

// ============================================================================
// Main - Register the plugin
// ============================================================================

func init() {
	// Create plugin instance
	p := &OrderlyPlugin{}

	// Register the plugin - this generates all standard WASM exports automatically
	// IMPORTANT: Must be in init(), not main(), so it runs before WASM exports are called
	plugin.RegisterPlugin(p)

	// Register stream handler - this generates handle_stream_message and handle_connection_event exports
	// The client implements the StreamHandler interface (HandleStreamMessage, HandleConnectionEvent)
	// NOTE: client will be initialized in OnInit, but we register it here so the exports are available
	// The actual client instance will be set when OnInit is called
	plugin.RegisterStreamHandler(&streamHandlerWrapper{p: p})
}

var log = logging.NewLogger("orderly-datasource")

func main() {
	// Required for WASM, but can be empty
}

// ============================================================================
// Plugin Implementation
// ============================================================================

// OrderlyPlugin implements the DataSourcePlugin interface
type OrderlyPlugin struct {
	config *plugin.ConfigStore
	client *orderly.Client
}

// GetMeta returns the plugin metadata. The version suffix and the
// AllowedNetworkTargets are build-time variants: `dev` builds add
// loopback targets for the Exchange Sandbox dialect (ORD-701), release
// builds keep the production hosts only.
func (p *OrderlyPlugin) GetMeta() m.Meta {
	return m.Meta{
		PluginID:    "orderly-datasource",
		Name:        "Orderly Network" + buildVariantName,
		AppID:       "datasrc",
		Category:    "dex",
		Description: "Orderly Network data source - perpetual futures markets",
		Author:      "trading_peter",
		Version:     "v1.3.0" + buildVariantVersion,
		Repository:  "https://github.com/trading-peter/plusev_datasource_orderly_plugin",
		Tags:        []string{"orderly", "crypto", "exchange", "perpetual", "futures"},
		Contacts: []m.AuthorContact{
			{
				Kind:  "x.com",
				Value: "https://x.com/trading_peter",
			},
		},
		Resources: m.ResourceAccess{
			AllowedNetworkTargets: networkTargets(),
		},
		// Features will be auto-populated with registered commands
		Features: []string{},
	}
}

// GetRateLimits returns an empty slice on purpose: Orderly's real budgets
// are declared as manifest Limits on the network targets (network.go), and
// keeping the legacy per-command limiter would double-throttle the same
// requests.
func (p *OrderlyPlugin) GetRateLimits() []plugin.RateLimit {
	return []plugin.RateLimit{}
}

// GetOrderCapabilities declares the order features this plugin really serves;
// the host checks every createOrders batch against it before dispatch
// (wasm export get_order_capabilities). It must mirror the create switch in
// createOrders.go exactly — an undeclared feature is rejected with a clear
// error, and a declared-but-ignored field would silently place a partial
// order.
//
// Not declared on purpose:
//   - TimeInForceGTD: Orderly's order/algo create bodies carry no expiry, so
//     ExpireTime cannot be sent.
//   - TrailingAmount: Orderly trailing stops are percentage-based only
//     (callbackRate/callbackValue).
//   - PositionSide / CancelGroupOnFailure: never read by the create path.
func (p *OrderlyPlugin) GetOrderCapabilities() ex.OrderCapabilities {
	return ex.OrderCapabilities{
		// Orderly publishes perpetual markets only (orderly/markets.go).
		AssetTypes: []tt.AssetType{tt.AssetTypePerpetual},
		OrderTypes: []acct.OrderType{
			acct.OrderTypeLimit, acct.OrderTypeMarket,
			acct.OrderTypeStop, acct.OrderTypeStopLimit, acct.OrderTypeStopMarket,
			acct.OrderTypeTakeProfit, acct.OrderTypeTakeProfitLimit, acct.OrderTypeTakeProfitMarket,
			acct.OrderTypeTrailingStop,
		},
		TimeInForce: []acct.TimeInForce{
			acct.TimeInForceGTC, acct.TimeInForceIOC, acct.TimeInForceFOK,
			acct.TimeInForcePostOnly, acct.TimeInForceGTX,
		},
		// Orderly trigger_price_type serves MARK_PRICE and LAST_PRICE; the
		// venue default (and this plugin's fallback) is the mark price.
		TriggerSources:       []acct.TriggerSource{acct.TriggerSourceMark, acct.TriggerSourceLast},
		DefaultTriggerSource: acct.TriggerSourceMark,

		TrailingPercent: true,
		ActivationPrice: true,
		ReduceOnly:      true,
		PostOnly:        true,
		ClosePosition:   true,
		QuoteQuantity:   true,
		ClientOrderID:   true,

		// Extra keys the create switch reads (createOrders.go); anything else
		// is rejected by the host. Both spellings are honored.
		ExtraKeys: []string{
			"visibleQuantity", "visible_quantity",
			"slippage",
			"level",
			"postOnlyAdjust", "post_only_adjust",
			"callbackRate",
			"callbackValue",
			"orderTag",
		},
	}
}

// The host reads the declaration through the plugin.OrderCapabilityProvider
// export; keep the wiring compiling.
var _ plugin.OrderCapabilityProvider = (*OrderlyPlugin)(nil)

// GetConfigFields returns the configuration fields needed by this plugin
func (p *OrderlyPlugin) GetConfigFields() []plugin.ConfigField {
	// Initialize client if needed to get config fields
	if p.client == nil {
		p.client = orderly.NewClient(requester.NewRequester(), urls.DefaultBaseURL, urls.DefaultWSBaseURL, false)
	}
	return p.client.GetConfigFields()
}

// OnInit is called when the plugin is initialized with user configuration
func (p *OrderlyPlugin) OnInit(config *plugin.ConfigStore) error {
	p.config = config

	// Resolve the hosts: explicit baseUrl/wsUrl win, then the testnet
	// toggle, then production (ORD-701).
	isTestnet := config.GetBool("testnet")
	baseURL, wsURL := urls.ResolveURLs(config.GetString("baseUrl"), config.GetString("wsUrl"), isTestnet)

	// Create Orderly client
	p.client = orderly.NewClient(requester.NewRequester(), baseURL, wsURL, isTestnet)

	// Set credentials on the client
	credentials := make(map[string]string)
	if accountID := config.GetString("accountID"); accountID != "" {
		credentials["accountID"] = accountID
	}
	if apiKey := config.GetString("apiKey"); apiKey != "" {
		credentials["apiKey"] = apiKey
	}
	if secretKey := config.GetString("secretKey"); secretKey != "" {
		credentials["secretKey"] = secretKey
	}

	p.client.SetCredentials(credentials)

	return nil
}

// OnShutdown is called when the plugin is being shut down
func (p *OrderlyPlugin) OnShutdown() error {
	// Cleanup resources if needed
	return nil
}

// RegisterCommands registers all command handlers
func (p *OrderlyPlugin) RegisterCommands(router *plugin.CommandRouter) {
	router.Register(ex.CMD_GET_MARKETS, p.handleGetMarkets)
	router.Register(ex.CMD_GET_TIMEFRAMES, p.handleGetTimeframes)
	router.Register(ex.CMD_OHLCV_STREAM, p.handleOHLCVStream)
	router.Register(ex.CMD_GET_OHLCV, p.handleGetOHLCV)
	router.Register(ex.CMD_GET_ACCOUNT, p.handleGetAccount)
	router.Register(ex.CMD_GET_BALANCES, p.handleGetBalances)
	router.Register(ex.CMD_GET_POSITIONS, p.handleGetPositions)
	router.Register(ex.CMD_GET_ORDERS, p.handleGetOrders)
	router.Register(ex.CMD_CREATE_ORDERS, p.handleCreateOrders)
	router.Register(ex.CMD_CANCEL_ORDERS, p.handleCancelOrders)
	router.Register("getRecentOHLCV", p.handleGetRecentOHLCV)
}

// ============================================================================
// Stream Handler Wrapper
// ============================================================================

// streamHandlerWrapper wraps the plugin to provide StreamHandler interface
// This allows us to register the stream handler before the client is initialized
type streamHandlerWrapper struct {
	p *OrderlyPlugin
}

func (w *streamHandlerWrapper) HandleStreamMessage(req plugin.StreamMessageRequest) (plugin.StreamMessageResponse, error) {
	if w.p.client == nil {
		log.ErrorWithData("streamHandlerWrapper.HandleStreamMessage: client not initialized", map[string]any{})
		return plugin.StreamMessageResponse{Success: false, Action: "ignore"}, fmt.Errorf("client not initialized")
	}

	response, err := w.p.client.HandleStreamMessage(req)

	if err != nil {
		log.ErrorWithData("streamHandlerWrapper.HandleStreamMessage client response", map[string]any{
			"response": response,
			"error":    err,
		})
	}

	return response, err
}

func (w *streamHandlerWrapper) HandleConnectionEvent(event plugin.StreamConnectionEvent) (plugin.StreamConnectionResponse, error) {
	if w.p.client == nil {
		log.ErrorWithData("streamHandlerWrapper.HandleConnectionEvent: client not initialized", map[string]any{})
		return plugin.StreamConnectionResponse{Success: false, Action: "ignore"}, fmt.Errorf("client not initialized")
	}

	response, err := w.p.client.HandleConnectionEvent(event)

	if err != nil {
		log.ErrorWithData("streamHandlerWrapper.HandleConnectionEvent client response", map[string]any{
			"response": response,
			"error":    err,
		})
	}

	return response, err
}
