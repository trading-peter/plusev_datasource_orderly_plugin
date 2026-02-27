package orderly

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/plusev-terminal/go-plugin-common/plugin"
	tt "github.com/plusev-terminal/go-plugin-common/trading"
)

// PrepareStream prepares streaming connection setup
func (c *Client) PrepareStream(request plugin.StreamSetupRequest) (plugin.StreamSetupResponse, error) {
	// Caller-owned context (required). This is the authoritative source for routing.
	streamContext, _ := request.Parameters["streamContext"].(map[string]any)
	if streamContext == nil {
		return plugin.StreamSetupResponse{Success: false, Error: "streamContext is required"}, nil
	}

	symbol, _ := streamContext["symbol"].(string)
	timeframe, _ := streamContext["timeframe"].(string)
	if strings.TrimSpace(symbol) == "" || strings.TrimSpace(timeframe) == "" {
		return plugin.StreamSetupResponse{Success: false, Error: "streamContext.symbol and streamContext.timeframe are required"}, nil
	}

	if c.accountID == "" {
		c.log.ErrorWithData("PrepareStream failed: missing account ID", map[string]any{})
		return plugin.StreamSetupResponse{
			Success: false,
			Error:   "Account ID is required for WebSocket connections. Please configure your Orderly credentials.",
		}, nil
	}

	// Convert timeframe to WebSocket topic format
	topicType := c.convertTimeframeToWebSocketTopic(timeframe)

	// Build WebSocket URL
	wsURL := fmt.Sprintf("%s/ws/stream/%s", c.wsBaseURL, c.accountID)

	// Create subscription message
	// Topic format: Try different formats based on common patterns
	// Let's try: PERP_SOL_USDC@kline_1m (Binance-style)
	topic := fmt.Sprintf("%s@%s", symbol, topicType)
	subscribeMsg := WSSubscribeMessage{
		ID:    "sub_1",
		Event: "subscribe",
		Topic: topic,
	}

	msgBytes, err := json.Marshal(subscribeMsg)
	if err != nil {
		c.log.ErrorWithData("PrepareStream failed to marshal subscription message", map[string]any{
			"error": err,
		})
		return plugin.StreamSetupResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to marshal subscribe message: %v", err),
		}, nil
	}

	initialMessages := []string{string(msgBytes)}

	response := plugin.StreamSetupResponse{
		Success:         true,
		WebSocketURL:    wsURL,
		Headers:         nil,
		Subprotocol:     "",
		InitialMessages: initialMessages,
		StreamContext:   streamContext,
	}

	return response, nil
}

// HandleStreamMessage processes incoming stream messages
func (c *Client) HandleStreamMessage(request plugin.StreamMessageRequest) (plugin.StreamMessageResponse, error) {
	// Try to parse as subscription response first
	var wsResponse WSResponse
	if err := json.Unmarshal(request.Message, &wsResponse); err == nil {
		if wsResponse.Event == "subscribe" && wsResponse.Success {
			// Subscription successful - ignore
			return plugin.StreamMessageResponse{
				Success: true,
				Action:  "ignore",
			}, nil
		}

		if wsResponse.Event == "subscribe" && !wsResponse.Success {
			return plugin.StreamMessageResponse{
				Success: true,
				Action:  "ignore",
			}, nil
		}
	}

	// Try to parse as kline update
	var klineUpdate WSKlineUpdate
	if err := json.Unmarshal(request.Message, &klineUpdate); err == nil {
		// Check if this is a kline topic (format: SYMBOL@kline_TIME)
		if strings.Contains(klineUpdate.Topic, "@kline_") {
			symbol, _ := request.StreamContext["symbol"].(string)
			timeframe, _ := request.StreamContext["timeframe"].(string)
			if symbol == "" || timeframe == "" {
				// Missing context - ignore message (host should persist StreamContext)
				return plugin.StreamMessageResponse{Success: true, Action: "ignore"}, nil
			}

			topicType := c.convertTimeframeToWebSocketTopic(timeframe)
			expectedTopic := fmt.Sprintf("%s@%s", symbol, topicType)

			// Only process messages that match this stream's symbol and timeframe
			if klineUpdate.Topic != expectedTopic {
				return plugin.StreamMessageResponse{
					Success: true,
					Action:  "ignore",
				}, nil
			}

			// Convert to OHLCV record
			// IsClosed is always false for live updates - Orderly doesn't provide this info,
			// consumers should use OHLCVDebouncer to detect closed candles
			record := tt.OHLCVRecord{
				OpenTime: klineUpdate.Data.StartTime / 1000, // Convert from milliseconds to seconds
				Open:     fmt.Sprintf("%.8f", klineUpdate.Data.Open),
				High:     fmt.Sprintf("%.8f", klineUpdate.Data.High),
				Low:      fmt.Sprintf("%.8f", klineUpdate.Data.Low),
				Close:    fmt.Sprintf("%.8f", klineUpdate.Data.Close),
				Volume:   fmt.Sprintf("%.8f", klineUpdate.Data.Volume),
				IsClosed: false,
			}

			response := plugin.StreamMessageResponse{
				Success:  true,
				Action:   "data",
				DataType: "ohlcv",
				Data:     record,
			}

			return response, nil
		}
	}

	return plugin.StreamMessageResponse{
		Success: true,
		Action:  "ignore",
	}, nil
}

// HandleConnectionEvent handles stream connection events
func (c *Client) HandleConnectionEvent(event plugin.StreamConnectionEvent) (plugin.StreamConnectionResponse, error) {
	switch event.EventType {
	case "connecting":
		return plugin.StreamConnectionResponse{
			Success: true,
			Action:  "ignore",
		}, nil
	case "connected":
		return plugin.StreamConnectionResponse{
			Success: true,
			Action:  "ignore",
		}, nil
	case "disconnected":
		return plugin.StreamConnectionResponse{
			Success: true,
			Action:  "reconnect",
		}, nil
	case "error":
		c.log.ErrorWithData("HandleConnectionEvent: WebSocket error occurred", map[string]any{
			"streamID": event.StreamID,
			"error":    event.Error,
		})
		return plugin.StreamConnectionResponse{
			Success: true,
			Action:  "ignore",
		}, nil
	default:
		return plugin.StreamConnectionResponse{
			Success: true,
			Action:  "ignore",
		}, nil
	}
}
