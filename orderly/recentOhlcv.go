package orderly

import (
	"fmt"

	"github.com/plusev-terminal/go-plugin-common/datasrc/exchange"
	rt "github.com/plusev-terminal/go-plugin-common/requester/types"
	tt "github.com/plusev-terminal/go-plugin-common/trading"
)

// GetRecentOHLCV fetches historical OHLCV data from Orderly using the private kline API
func (c *Client) GetRecentOHLCV(params exchange.GetOHLCVParams) ([]tt.OHLCVRecord, error) {
	symbol := params.Market.Symbol
	if symbol == "" {
		return nil, fmt.Errorf("market.symbol is required")
	}
	// Map timeframe to Orderly type
	// Orderly supports: 1m/5m/15m/30m/1h/4h/12h/1d/1w/1mon/1y
	var orderlyType string
	switch params.Timeframe {
	case "1M":
		orderlyType = "1mon"
	default:
		orderlyType = params.Timeframe
	}

	// Set limit with default
	limit := params.Limit
	if limit <= 0 {
		limit = DefaultLimit
	}

	// Cap the limit to prevent excessive API calls
	if limit > OrderlyMaxLimit {
		limit = OrderlyMaxLimit
	}

	// Build query parameters
	// Note: This endpoint is private and requires authentication
	queryParams := fmt.Sprintf("symbol=%s&type=%s&limit=%d", symbol, orderlyType, limit)
	endpoint := c.baseURL + "/v1/kline?" + queryParams

	req := &rt.Request{
		Method: "GET",
		URL:    endpoint,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	// Add authentication headers (REQUIRED for private endpoint)
	c.addAuthHeaders(req, "")

	var response KlineResponse
	_, err := c.requester.Send(req, &response)
	if err != nil {
		c.log.ErrorWithData("Failed to fetch recent OHLCV data from Orderly", map[string]any{
			"error":    err.Error(),
			"endpoint": endpoint,
		})
		return nil, fmt.Errorf("failed to fetch recent OHLCV data from Orderly code=%d message=%s: %w", response.Code, response.Message, err)
	}

	if !response.Success {
		return nil, fmt.Errorf("Orderly API returned success=false code=%d message=%s", response.Code, response.Message)
	}

	// Convert Orderly format to our OHLCV format
	var records []tt.OHLCVRecord
	for _, kline := range response.Data.Rows {
		records = append(records, tt.OHLCVRecord{
			OpenTime: kline.StartTime / 1000, // Convert from milliseconds to seconds
			Open:     fmt.Sprintf("%.8f", kline.Open),
			High:     fmt.Sprintf("%.8f", kline.High),
			Low:      fmt.Sprintf("%.8f", kline.Low),
			Close:    fmt.Sprintf("%.8f", kline.Close),
			Volume:   fmt.Sprintf("%.8f", kline.Volume),
		})
	}

	return records, nil
}
