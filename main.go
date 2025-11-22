package main

import (
	"fmt"
	"strings"
	"time"

	ex "github.com/plusev-terminal/go-plugin-common/datasrc/exchange"
	"github.com/plusev-terminal/go-plugin-common/logging"
	m "github.com/plusev-terminal/go-plugin-common/meta"
	"github.com/plusev-terminal/go-plugin-common/plugin"
	"github.com/plusev-terminal/go-plugin-common/requester"
	commonstream "github.com/plusev-terminal/go-plugin-common/stream"

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
		Version:     "v1.0.0",
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
	if privateKey := config.GetString("privateKey"); privateKey != "" {
		credentials["privateKey"] = privateKey
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
	router.Register("getRecentOHLCV", p.handleGetRecentOHLCV)
}

// ============================================================================
// Command Handlers
// ============================================================================

// handleGetMarkets returns available trading pairs
func (p *OrderlyPlugin) handleGetMarkets(params map[string]any) plugin.Response {
	markets, err := p.client.GetMarkets()
	if err != nil {
		return plugin.ErrorResponse(err)
	}

	return plugin.SuccessResponse(markets, time.Hour*12)
}

// handleGetTimeframes returns supported timeframes
func (p *OrderlyPlugin) handleGetTimeframes(params map[string]any) plugin.Response {
	timeframes := p.client.GetTimeframes()
	return plugin.SuccessResponse(timeframes, time.Hour*12)
}

// handleGetOHLCV returns historical OHLCV data
func (p *OrderlyPlugin) handleGetOHLCV(params map[string]any) plugin.Response {
	// Safety check
	if p.client == nil {
		return plugin.ErrorResponseMsg("client not initialized")
	}

	// Extract validated parameters - validation already done by terminal
	ohlcvParams := ex.GetOHLCVParamsFromMap(params)

	// Fetch OHLCV data
	records, err := p.client.GetOHLCV(ohlcvParams)
	if err != nil {
		log.ErrorWithData("handleGetOHLCV error", map[string]any{
			"error": err.Error(),
		})
		return plugin.ErrorResponse(err)
	}

	if ohlcvParams.CacheForSeconds > 0 {
		cacheDuration := time.Duration(ohlcvParams.CacheForSeconds) * time.Second
		return plugin.SuccessResponse(records, cacheDuration)
	}

	return plugin.SuccessResponse(records)
}

// handleGetOHLCV returns recent OHLCV data which is capped less history and doesn't support start/end time.
func (p *OrderlyPlugin) handleGetRecentOHLCV(params map[string]any) plugin.Response {
	// Safety check
	if p.client == nil {
		return plugin.ErrorResponseMsg("client not initialized")
	}

	// Extract validated parameters - validation already done by terminal
	ohlcvParams := ex.GetOHLCVParamsFromMap(params)

	// Fetch OHLCV data
	records, err := p.client.GetRecentOHLCV(ohlcvParams)
	if err != nil {
		log.ErrorWithData("handleGetRecentOHLCV error", map[string]any{
			"error": err.Error(),
		})
		return plugin.ErrorResponse(err)
	}

	if ohlcvParams.CacheForSeconds > 0 {
		cacheDuration := time.Duration(ohlcvParams.CacheForSeconds) * time.Second
		return plugin.SuccessResponse(records, cacheDuration)
	}

	return plugin.SuccessResponse(records)
}

// handleOHLCVStream sets up a WebSocket stream for OHLCV data
func (p *OrderlyPlugin) handleOHLCVStream(params map[string]any) plugin.Response {
	// Extract validated parameters - validation already done by terminal
	streamParams := ex.OHLCVStreamParamsFromMap(params)
	if strings.TrimSpace(streamParams.Market.Symbol) == "" {
		return plugin.ErrorResponseMsg("market.symbol is required")
	}
	assetType := strings.TrimSpace(streamParams.Market.AssetType)

	// Prepare stream setup request
	streamReq := plugin.StreamSetupRequest{
		StreamID:   fmt.Sprintf("orderly_ohlcv_%s_%s", streamParams.Market.Symbol, streamParams.Timeframe),
		StreamType: "ohlcv",
		Parameters: map[string]any{
			"market":    streamParams.Market,
			"timeframe": streamParams.Timeframe,
			"private":   false, // Public stream for OHLCV
			"streamContext": map[string]any{
				"symbol":    streamParams.Market.Symbol,
				"timeframe": streamParams.Timeframe,
				"assetType": assetType,
			},
		},
	}

	// Get stream setup from client
	setupResp, err := p.client.PrepareStream(streamReq)
	if err != nil {
		log.ErrorWithData("handleOHLCVStream PrepareStream failed", map[string]any{
			"error": err,
		})
		return plugin.ErrorResponse(err)
	}

	if !setupResp.Success {
		log.ErrorWithData("handleOHLCVStream setup response failed", map[string]any{
			"error": setupResp.Error,
		})
		return plugin.ErrorResponseMsg(setupResp.Error)
	}

	// Return stream marker for the datasrc system to handle
	marker := commonstream.StreamMarker{
		Stream:          true,
		StreamID:        streamReq.StreamID,
		WebSocketURL:    setupResp.WebSocketURL,
		Headers:         setupResp.Headers,
		Subprotocol:     setupResp.Subprotocol,
		InitialMessages: setupResp.InitialMessages,
		StreamContext:   setupResp.StreamContext,
		Heartbeat: &commonstream.StreamHeartbeatSpec{
			App: &commonstream.AppHeartbeatSpec{
				MatchJSONField:       "event",
				PingValue:            "ping",
				PongValue:            "pong",
				ClientPingIntervalMs: 10000,
			},
		},
	}

	if err := marker.Validate(); err != nil {
		return plugin.ErrorResponseMsg(err.Error())
	}
	if marker.Heartbeat != nil && marker.Heartbeat.App != nil {
		if err := marker.Heartbeat.App.Validate(); err != nil {
			return plugin.ErrorResponseMsg(err.Error())
		}
	}

	return plugin.SuccessTypedResponse("StreamMarker", marker)
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
