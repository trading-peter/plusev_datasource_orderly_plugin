package orderly

import (
	"fmt"
	"strconv"
	"strings"

	rt "github.com/plusev-terminal/go-plugin-common/requester/types"
	tt "github.com/plusev-terminal/go-plugin-common/trading"
)

// GetMarkets returns all available trading markets from Orderly
func (c *Client) GetMarkets() ([]tt.Market, error) {
	req := &rt.Request{
		Method: "GET",
		URL:    c.baseURL + "/v1/public/info",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	var response SymbolsResponse
	_, err := c.requester.Send(req, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch symbols from Orderly code=%d message=%s: %w", response.Code, response.Message, err)
	}

	if !response.Success {
		return nil, fmt.Errorf("Orderly API returned success=false code=%d message=%s", response.Code, response.Message)
	}

	var markets []tt.Market
	for _, symbol := range response.Data.Rows {
		// Orderly symbols are typically like "PERP_BTC_USDC"
		// Split to extract base and quote
		parts := strings.Split(symbol.Symbol, "_")
		if len(parts) < 3 {
			continue // Skip malformed symbols
		}

		// Assume all Orderly markets are perpetual futures for now
		assetType := "perpetual"
		base := parts[1]  // BTC
		quote := parts[2] // USDC

		// Create human-readable label
		label := fmt.Sprintf("%s-PERP", base)

		// Use string conversion to avoid WASM precision issues
		priceTickStr := fmt.Sprintf("%.15g", symbol.QuoteTick)
		qtyTickStr := fmt.Sprintf("%.15g", symbol.BaseTick)
		pricePrecision := countPrecisionFromTickString(priceTickStr)
		qtyPrecision := countPrecisionFromTickString(qtyTickStr)

		markets = append(markets, tt.Market{
			Label:     label,
			Symbol:    symbol.Symbol,
			Base:      base,
			Quote:     quote,
			AssetType: assetType,

			// Precision & limits
			PriceTick:         priceTickStr,
			QuantityTick:      qtyTickStr,
			PricePrecision:    pricePrecision,
			QuantityPrecision: qtyPrecision,
			MinQuantity:       fmt.Sprintf("%.15g", symbol.BaseMin),
			MaxQuantity:       fmt.Sprintf("%.15g", symbol.BaseMax),
			MinNotional:       fmt.Sprintf("%.15g", symbol.MinNotional),
			MaxNotional:       fmt.Sprintf("%.15g", symbol.QuoteMax),

			// Fees & rates
			LiquidationFee: fmt.Sprintf("%.15g", symbol.StdLiquidationFee),

			// Leverage & margin
			InitialMarginRate:     fmt.Sprintf("%.15g", symbol.BaseImr),
			MaintenanceMarginRate: fmt.Sprintf("%.15g", symbol.BaseMmr),

			// Funding
			FundingInterval: symbol.FundingPeriod,
			FundingCap:      fmt.Sprintf("%.15g", symbol.CapFunding),
			FundingFloor:    fmt.Sprintf("%.15g", symbol.FloorFunding),
		})
	}

	return markets, nil
}

func countPrecisionFromTickString(tick string) int {
	if tick == "" {
		return 0
	}
	// Normalize scientific notation to a plain decimal string when possible.
	if strings.ContainsAny(tick, "eE") {
		if f, err := strconv.ParseFloat(tick, 64); err == nil {
			// Use a high precision fixed format, then trim.
			tick = strconv.FormatFloat(f, 'f', 18, 64)
		}
	}
	if tick == "0" {
		return 0
	}
	idx := strings.Index(tick, ".")
	if idx == -1 {
		return 0
	}
	dec := strings.TrimRight(tick[idx+1:], "0")
	if dec == "" {
		return 0
	}
	// Practical cap for UI usage.
	if len(dec) > 12 {
		return 12
	}
	return len(dec)
}

// GetTimeframes returns the timeframes supported by Orderly
func (c *Client) GetTimeframes() []tt.Timeframe {
	// Based on Orderly WebSocket API documentation: 1m/5m/15m/30m/1h/1d/1w/1M
	return []tt.Timeframe{
		{Value: 1, Unit: tt.Minutes},
		{Value: 3, Unit: tt.Minutes},
		{Value: 5, Unit: tt.Minutes},
		{Value: 15, Unit: tt.Minutes},
		{Value: 30, Unit: tt.Minutes},
		{Value: 1, Unit: tt.Hours},
		{Value: 4, Unit: tt.Hours},
		{Value: 12, Unit: tt.Hours},
		{Value: 1, Unit: tt.Days},
		{Value: 1, Unit: tt.Weeks},
		{Value: 1, Unit: tt.Months},
	}
}

// convertTimeframeToTradingViewResolution converts our timeframe format to TradingView resolution
func (c *Client) convertTimeframeToTradingViewResolution(timeframe string) string {
	tf, err := tt.TimeframeFromString(timeframe)
	if err != nil {
		c.log.ErrorWithData("Invalid timeframe format", map[string]any{"timeframe": timeframe, "error": err})
		return "1h" // Default to 1 hour
	}

	switch tf.Unit {
	case tt.Minutes:
		return fmt.Sprintf("%dm", int(tf.Value))
	case tt.Hours:
		return fmt.Sprintf("%dh", int(tf.Value))
	case tt.Days:
		return fmt.Sprintf("%dd", int(tf.Value))
	case tt.Weeks:
		return fmt.Sprintf("%dw", int(tf.Value))
	case tt.Months:
		return fmt.Sprintf("%dmon", int(tf.Value))
	}

	c.log.ErrorWithData("Unsupported timeframe format", map[string]any{"timeframe": timeframe})
	return "1h" // Default to 1 hour
}

// convertTimeframeToWebSocketTopic converts our timeframe format to WebSocket topic format
func (c *Client) convertTimeframeToWebSocketTopic(timeframe string) string {
	switch timeframe {
	case "1m":
		return "kline_1m"
	case "5m":
		return "kline_5m"
	case "15m":
		return "kline_15m"
	case "30m":
		return "kline_30m"
	case "1h":
		return "kline_1h"
	case "1d":
		return "kline_1d"
	case "1w":
		return "kline_1w"
	case "1M":
		return "kline_1M"
	default:
		return "kline_1h" // Default to 1 hour
	}
}
