package orderly

const (
	// OrderlyMaxLimit is the maximum number of candles that can be fetched in a single request
	OrderlyMaxLimit = 1000
	// DefaultLimit is the default number of candles to fetch when limit is not specified
	DefaultLimit = 100
)

// OrderlyResponse represents a generic Orderly API response
type OrderlyResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Code      int         `json:"code,omitempty"`
	Message   string      `json:"message,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

// Symbol represents an Orderly trading symbol/market
type Symbol struct {
	Symbol                     string  `json:"symbol"`
	QuoteMin                   float64 `json:"quote_min"`
	QuoteMax                   float64 `json:"quote_max"`
	QuoteTick                  float64 `json:"quote_tick"`
	BaseMin                    float64 `json:"base_min"`
	BaseMax                    float64 `json:"base_max"`
	BaseTick                   float64 `json:"base_tick"`
	MinNotional                float64 `json:"min_notional"`
	PriceRange                 float64 `json:"price_range"`
	PriceScope                 float64 `json:"price_scope"`
	StdLiquidationFee          float64 `json:"std_liquidation_fee"`
	LiquidatorFee              float64 `json:"liquidator_fee"`
	ClaimInsuranceFundDiscount float64 `json:"claim_insurance_fund_discount"`
	FundingPeriod              int     `json:"funding_period"`
	CapFunding                 float64 `json:"cap_funding"`
	FloorFunding               float64 `json:"floor_funding"`
	CapIr                      float64 `json:"cap_ir"`
	FloorIr                    float64 `json:"floor_ir"`
	InterestRate               float64 `json:"interest_rate"`
	ImrFactor                  float64 `json:"imr_factor"`
	CreatedTime                int64   `json:"created_time"`
	UpdatedTime                int64   `json:"updated_time"`
	BaseMmr                    float64 `json:"base_mmr"`
	BaseImr                    float64 `json:"base_imr"`
	LiquidationTier            int     `json:"liquidation_tier"`
	GlobalMaxOiCap             float64 `json:"global_max_oi_cap"`
}

// SymbolsResponse represents the response from the symbols endpoint
type SymbolsResponse struct {
	Success   bool  `json:"success"`
	Timestamp int64 `json:"timestamp"`
	Data      struct {
		Rows []Symbol `json:"rows"`
	} `json:"data"`
}

// TradingViewHistoryResponse represents the response from TradingView history API
type TradingViewHistoryResponse struct {
	S string    `json:"s"` // Status: "ok" or "no_data"
	T []int64   `json:"t"` // Timestamps
	O []float64 `json:"o"` // Open prices
	H []float64 `json:"h"` // High prices
	L []float64 `json:"l"` // Low prices
	C []float64 `json:"c"` // Close prices
	V []float64 `json:"v"` // Volume
}

// KlineData represents a single kline/candle data point from REST API
type KlineData struct {
	Symbol    string  `json:"symbol"`
	Type      string  `json:"type"`
	Open      float64 `json:"open"`
	Close     float64 `json:"close"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Volume    float64 `json:"volume"`
	Amount    float64 `json:"amount"`
	StartTime int64   `json:"start_timestamp"`
	EndTime   int64   `json:"end_timestamp"`
}

// KlineResponse represents the response from the kline endpoint
type KlineResponse struct {
	Success   bool  `json:"success"`
	Timestamp int64 `json:"timestamp"`
	Data      struct {
		Rows []KlineData `json:"rows"`
	} `json:"data"`
}

// PositionsResponse represents the response from GET /v1/positions.
type PositionsResponse struct {
	Success   bool  `json:"success"`
	Timestamp int64 `json:"timestamp"`
	Data      struct {
		CurrentMarginRatioWithOrders     float64       `json:"current_margin_ratio_with_orders"`
		FreeCollateral                   float64       `json:"free_collateral"`
		InitialMarginRatio               float64       `json:"initial_margin_ratio"`
		InitialMarginRatioWithOrders     float64       `json:"initial_margin_ratio_with_orders"`
		MaintenanceMarginRatio           float64       `json:"maintenance_margin_ratio"`
		MaintenanceMarginRatioWithOrders float64       `json:"maintenance_margin_ratio_with_orders"`
		MarginRatio                      float64       `json:"margin_ratio"`
		OpenMarginRatio                  float64       `json:"open_margin_ratio"`
		TotalCollateralValue             float64       `json:"total_collateral_value"`
		TotalPnl24H                      float64       `json:"total_pnl_24_h"`
		Rows                             []PositionRow `json:"rows"`
	} `json:"data"`
}

// PositionRow represents a position item in PositionsResponse.Data.Rows.
type PositionRow struct {
	Symbol                string  `json:"symbol"`
	PositionQty           float64 `json:"position_qty"`
	AverageOpenPrice      float64 `json:"average_open_price"`
	MarkPrice             float64 `json:"mark_price"`
	EstLiqPrice           float64 `json:"est_liq_price"`
	UnsettledPnl          float64 `json:"unsettled_pnl"`
	Pnl24H                float64 `json:"pnl_24_h"`
	Fee24H                float64 `json:"fee_24_h"`
	Leverage              string  `json:"leverage"`
	CostPosition          float64 `json:"cost_position"`
	SettlePrice           float64 `json:"settle_price"`
	PendingLongQty        float64 `json:"pending_long_qty"`
	PendingShortQty       float64 `json:"pending_short_qty"`
	Timestamp             int64   `json:"timestamp"`
	UpdatedTime           int64   `json:"updated_time"`
	Seq                   int64   `json:"seq"`
	Imr                   float64 `json:"imr"`
	Mmr                   float64 `json:"mmr"`
	IMRWithdrawOrders     float64 `json:"IMR_withdraw_orders"`
	MMRWithOrders         float64 `json:"MMR_with_orders"`
	LastSumUnitaryFunding float64 `json:"last_sum_unitary_funding"`
}

// WebSocket message structures

// WSSubscribeMessage represents a WebSocket subscription message
type WSSubscribeMessage struct {
	ID    string `json:"id"`
	Event string `json:"event"` // "subscribe" or "unsubscribe"
	Topic string `json:"topic"` // Format: kline_1m:PERP_BTC_USDC
}

// WSResponse represents a generic WebSocket response
type WSResponse struct {
	ID      string `json:"id"`
	Event   string `json:"event"`
	Success bool   `json:"success"`
	Ts      int64  `json:"ts"`
	Data    any    `json:"data,omitempty"`
}

// WSKlineUpdate represents a WebSocket kline update message
type WSKlineUpdate struct {
	Topic string      `json:"topic"`
	Ts    int64       `json:"ts"`
	Data  WSKlineData `json:"data"`
}

// WSKlineData represents kline data in a WebSocket message
type WSKlineData struct {
	Symbol    string  `json:"symbol"`
	Type      string  `json:"type"`
	Open      float64 `json:"open"`
	Close     float64 `json:"close"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Volume    float64 `json:"volume"`
	Amount    float64 `json:"amount"`
	StartTime int64   `json:"startTime"`
	EndTime   int64   `json:"endTime"`
}
