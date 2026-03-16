package main

import (
	"fmt"
	"time"

	ex "github.com/plusev-terminal/go-plugin-common/datasrc/exchange"
	"github.com/plusev-terminal/go-plugin-common/logging"
	m "github.com/plusev-terminal/go-plugin-common/meta"
	"github.com/plusev-terminal/go-plugin-common/plugin"
	"github.com/plusev-terminal/go-plugin-common/requester"
	"github.com/trading-peter/plusev_datasource_orderly_plugin/orderly"
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

// GetMeta returns the plugin metadata
func (p *OrderlyPlugin) GetMeta() m.Meta {
	return m.Meta{
		PluginID:    "orderly-datasource",
		Name:        "Orderly Network",
		AppID:       "datasrc",
		Category:    "dex",
		Description: "Orderly Network data source - perpetual futures markets",
		Author:      "trading_peter",
		Version:     "v1.2.1",
		Repository:  "https://github.com/trading-peter/plusev_datasource_orderly_plugin",
		Tags:        []string{"orderly", "crypto", "exchange", "perpetual", "futures"},
		Contacts: []m.AuthorContact{
			{
				Kind:  "x.com",
				Value: "https://x.com/trading_peter",
			},
		},
		Resources: m.ResourceAccess{
			AllowedNetworkTargets: []m.NetworkTargetRule{
				{Pattern: "https://api.orderly.org/*"},
				{Pattern: "https://testnet-api.orderly.org/*"},
				{Pattern: "wss://ws-evm.orderly.org/*"},
				{Pattern: "wss://testnet-ws-evm.orderly.org/*"},
			},
		},
		// Features will be auto-populated with registered commands
		Features: []string{},
	}
}

func (p *OrderlyPlugin) GetRateLimits() []plugin.RateLimit {
	// Define rate limits based on Orderly API documentation
	return []plugin.RateLimit{
		// Public endpoints
		{
			Command: ex.CMD_GET_MARKETS, // GET /v1/public/info
			Scope:   []plugin.RateLimitScope{plugin.RateLimitScopeIP},
			RPS:     10.0, // Conservative rate limit
			Burst:   10,
		},
		{
			Command: ex.CMD_GET_TIMEFRAMES, // Static data, no API call
			Scope:   []plugin.RateLimitScope{plugin.RateLimitScopeIP},
			RPS:     10.0,
			Burst:   10,
		},
		// Private endpoints (require authentication)
		{
			Command: ex.CMD_GET_OHLCV, // GET /v1/tv/kline_history
			Scope:   []plugin.RateLimitScope{plugin.RateLimitScopeAPIKey},
			RPS:     plugin.CalculateRPS(5, 10*time.Second), // 5 requests per 10 seconds as per API docs
			Burst:   1,
		},
		{
			Command: ex.CMD_CREATE_ORDERS, // POST /v1/order or /v1/batch-order or /v1/algo/order
			Scope:   []plugin.RateLimitScope{plugin.RateLimitScopeAPIKey},
			RPS:     1.0, // Conservative (batch create limit is 1 rps)
			Burst:   2,
		},
		// WebSocket streams
		{
			Command: ex.CMD_OHLCV_STREAM,
			Scope:   []plugin.RateLimitScope{plugin.RateLimitScopeAPIKey},
			RPS:     1.0, // Connection setup rate
			Burst:   5,   // Allow some burst for reconnections
		},
	}
}

// GetConfigFields returns the configuration fields needed by this plugin
func (p *OrderlyPlugin) GetConfigFields() []plugin.ConfigField {
	// Initialize client if needed to get config fields
	if p.client == nil {
		p.client = orderly.NewClient(requester.NewRequester(), "https://api.orderly.org", false)
	}
	return p.client.GetConfigFields()
}

// OnInit is called when the plugin is initialized with user configuration
func (p *OrderlyPlugin) OnInit(config *plugin.ConfigStore) error {
	p.config = config

	// Determine if testnet should be used
	isTestnet := config.GetBool("testnet")
	baseURL := "https://api.orderly.org"
	if isTestnet {
		baseURL = "https://testnet-api.orderly.org"
	}

	// Create Orderly client
	p.client = orderly.NewClient(requester.NewRequester(), baseURL, isTestnet)

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
