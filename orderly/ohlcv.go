package orderly

import (
	"fmt"

	"github.com/plusev-terminal/go-plugin-common/datasrc/exchange"
	rt "github.com/plusev-terminal/go-plugin-common/requester/types"
	tt "github.com/plusev-terminal/go-plugin-common/trading"
	utils "github.com/plusev-terminal/go-plugin-common/wasmutils"
)

// GetOHLCV fetches historical OHLCV data from Orderly using /v1/tv/kline_history endpoint
func (c *Client) GetOHLCV(params exchange.GetOHLCVParams) ([]tt.OHLCVRecord, error) {
	symbol := params.Market.Symbol
	if symbol == "" {
		return nil, fmt.Errorf("market.symbol is required")
	}
	// Convert timeframe to resolution format
	resolution := c.convertTimeframeToTradingViewResolution(params.Timeframe)

	// Parse timeframe to get duration for calculations
	tf, err := tt.TimeframeFromString(params.Timeframe)
	if err != nil {
		c.log.ErrorWithData("Invalid timeframe format", map[string]any{"timeframe": params.Timeframe, "error": err})
		return nil, fmt.Errorf("invalid timeframe format: %w", err)
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

	// Calculate time range based on the different scenarios
	now, _ := utils.Now()
	var from, to int64

	switch {
	case params.StartTime == nil && params.EndTime == nil:
		// Case 1: Both nil - serve "limit" candles back from now
		to = now.Unix()
		candleDurationMinutes := tf.ToMinutes()
		from = to - int64(limit*candleDurationMinutes*60) // Convert to seconds

	case params.StartTime != nil && params.EndTime == nil:
		// Case 2: start not nil, end nil - serve "limit" candles forward from "start"
		from = params.StartTime.Unix()
		candleDurationMinutes := tf.ToMinutes()
		to = from + int64(limit*candleDurationMinutes*60) // Convert to seconds

	case params.StartTime == nil && params.EndTime != nil:
		// Case 3: start nil, end not nil - serve "limit" candles back starting at "end"
		to = params.EndTime.Unix()
		candleDurationMinutes := tf.ToMinutes()
		from = to - int64(limit*candleDurationMinutes*60) // Convert to seconds

	case params.StartTime != nil && params.EndTime != nil:
		// Case 4: both not nil - serve candles from start to end, capped by max limit
		from = params.StartTime.Unix()
		to = params.EndTime.Unix()

		// Calculate how many candles this time range represents
		durationSeconds := to - from
		candleDurationMinutes := tf.ToMinutes()
		candleDurationSeconds := int64(candleDurationMinutes * 60)

		if candleDurationSeconds > 0 {
			estimatedCandles := int(durationSeconds / candleDurationSeconds)

			// If the time range would result in more candles than our max limit, cap it
			if estimatedCandles > OrderlyMaxLimit {
				to = from + int64(OrderlyMaxLimit*candleDurationMinutes*60)
				c.log.WarnWithData("Time range capped to max limit", map[string]any{
					"originalTo":       params.EndTime.Unix(),
					"cappedTo":         to,
					"estimatedCandles": estimatedCandles,
					"maxLimit":         OrderlyMaxLimit,
				})
			}
		}
	}

	// Build query parameters
	queryParams := fmt.Sprintf("symbol=%s&resolution=%s&from=%d&to=%d", symbol, resolution, from, to)
	endpoint := c.baseURL + "/v1/tv/kline_history?" + queryParams

	req := &rt.Request{
		Method: "GET",
		URL:    endpoint,
	}

	// Add authentication headers (REQUIRED for this endpoint)
	c.addAuthHeaders(req, "")

	var response TradingViewHistoryResponse
	_, err = c.requester.Send(req, &response)
	if err != nil {
		c.log.ErrorWithData("Failed to fetch OHLCV data from Orderly", map[string]any{
			"error":    err.Error(),
			"endpoint": endpoint,
		})
		return nil, fmt.Errorf("failed to fetch OHLCV data from Orderly: %w", err)
	}

	if response.S != "ok" {
		if response.S == "no_data" {
			return []tt.OHLCVRecord{}, nil // Return empty slice for no data
		}
		return nil, fmt.Errorf("Orderly TradingView API returned status: %s", response.S)
	}

	// Convert TradingView format to our OHLCV format
	var records []tt.OHLCVRecord
	for i := 0; i < len(response.T); i++ {
		// Apply limit if specified (final safety check)
		if params.Limit > 0 && len(records) >= params.Limit {
			break
		}

		records = append(records, tt.OHLCVRecord{
			OpenTime: response.T[i],
			Open:     fmt.Sprintf("%.8f", response.O[i]),
			High:     fmt.Sprintf("%.8f", response.H[i]),
			Low:      fmt.Sprintf("%.8f", response.L[i]),
			Close:    fmt.Sprintf("%.8f", response.C[i]),
			Volume:   fmt.Sprintf("%.8f", response.V[i]),
			IsClosed: true,
		})
	}

	return records, nil
}
