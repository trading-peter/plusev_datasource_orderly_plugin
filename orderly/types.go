package orderly

import "encoding/json"

const (
	// OrderlyMaxLimit is the maximum number of candles that can be fetched in a single request
	OrderlyMaxLimit = 1000
	// DefaultLimit is the default number of candles to fetch when limit is not specified
	DefaultLimit = 100
)

// OrderlyResponse represents a generic Orderly API response
type OrderlyResponse struct {
	Success   bool   `json:"success"`
	Code      int    `json:"code,omitzero"`
	Message   string `json:"message,omitzero"`
	Timestamp int64  `json:"timestamp"`
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
	OrderlyResponse
	Data struct {
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
	OrderlyResponse
	Data struct {
		Rows []KlineData `json:"rows"`
	} `json:"data"`
}

type ClientInfoResponse struct {
	OrderlyResponse
	Data struct {
		AccountID   string  `json:"account_id"`
		Email       string  `json:"email"`
		AccountMode string  `json:"account_mode"`
		MaxLeverage float64 `json:"max_leverage"`

		MaintenanceCancelOrders bool           `json:"maintenance_cancel_orders"`
		ImrFactor               map[string]any `json:"imr_factor"`
		MaxNotional             map[string]any `json:"max_notional"`
		Extra                   map[string]any `json:"-"`
	} `json:"data"`
}

// PositionsResponse represents the response from GET /v1/positions.
type PositionsResponse struct {
	OrderlyResponse
	Data struct {
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

type HoldingItem struct {
	UpdatedTime  int64   `json:"updated_time"`
	Token        string  `json:"token"`
	Holding      float64 `json:"holding"`
	Frozen       float64 `json:"frozen"`
	PendingShort float64 `json:"pending_short"`
}

type ClientHoldingResponse struct {
	Success   bool  `json:"success"`
	Timestamp int64 `json:"timestamp"`
	Data      struct {
		Holding []HoldingItem `json:"holding"`
	} `json:"data"`
}

// OrdersResponse represents the response from GET /v1/orders.
type OrdersResponse struct {
	OrderlyResponse
	Data struct {
		Rows []OrderRow `json:"rows"`
	} `json:"data"`
}

// OrderRow represents an order in OrdersResponse.Data.Rows.
// Fields are kept permissive to match Orderly API variations.
type OrderRow struct {
	OrderID       StringOrNumber `json:"order_id"`
	Symbol        string         `json:"symbol"`
	Side          string         `json:"side"`
	OrderType     string         `json:"order_type"`
	Price         float64        `json:"price"`
	Quantity      float64        `json:"quantity"`
	ExecutedQty   float64        `json:"executed_qty"`
	Status        string         `json:"status"`
	CreatedTime   int64          `json:"created_time"`
	UpdatedTime   int64          `json:"updated_time"`
	ClientOrderID string         `json:"client_order_id"`
}

// AlgoOrdersResponse represents the response from GET /v1/algo/orders.
type AlgoOrdersResponse struct {
	OrderlyResponse
	Data struct {
		Rows []AlgoOrderRow `json:"rows"`
	} `json:"data"`
}

// AlgoOrderRow represents an algo order (STOP, TPSL, BRACKET, etc.).
type AlgoOrderRow struct {
	AlgoOrderID           StringOrNumber `json:"algo_order_id"`
	RootAlgoOrderID       StringOrNumber `json:"root_algo_order_id"`
	ParentAlgoOrderID     StringOrNumber `json:"parent_algo_order_id"`
	Symbol                string         `json:"symbol"`
	Side                  string         `json:"side"`
	AlgoType              string         `json:"algo_type"`   // STOP, TPSL, TP_SL, BRACKET, etc
	AlgoStatus            string         `json:"algo_status"` // NEW, CANCELLED, FILLED, etc
	RootAlgoOrderStatus   string         `json:"root_algo_order_status"`
	Type                  string         `json:"type"` // LIMIT, MARKET
	Status                string         `json:"status"`
	Quantity              float64        `json:"quantity"`
	ExecutedQuantity      float64        `json:"executed_quantity"`
	TotalExecutedQuantity float64        `json:"total_executed_quantity"`
	AverageExecutedPrice  float64        `json:"average_executed_price"`
	VisibleQuantity       float64        `json:"visible_quantity"`
	Price                 float64        `json:"price"`
	TriggerPrice          float64        `json:"trigger_price"`
	TriggerStatus         string         `json:"trigger_status"`
	TriggerPriceType      string         `json:"trigger_price_type"` // MARK_PRICE, LAST_PRICE
	TriggerTime           int64          `json:"trigger_time"`
	IsTriggered           bool           `json:"is_triggered"`
	IsActivated           bool           `json:"is_activated"`
	ReduceOnly            bool           `json:"reduce_only"`
	TotalFee              float64        `json:"total_fee"`
	FeeAsset              string         `json:"fee_asset"`
	RealizedPnl           float64        `json:"realized_pnl"`
	OrderTag              string         `json:"order_tag"`
	CreatedTime           int64          `json:"created_time"`
	UpdatedTime           int64          `json:"updated_time"`
	ChildOrders           []AlgoOrderRow `json:"child_orders,omitempty"`
}

// CreateOrderRequest represents the request body for POST /v1/order.
type CreateOrderRequest struct {
	Symbol          string   `json:"symbol"`
	OrderType       string   `json:"order_type"`
	Side            string   `json:"side"`
	ClientOrderID   string   `json:"client_order_id,omitempty"`
	OrderPrice      *float64 `json:"order_price,omitempty"`
	OrderQuantity   *float64 `json:"order_quantity,omitempty"`
	OrderAmount     *float64 `json:"order_amount,omitempty"`
	VisibleQuantity *float64 `json:"visible_quantity,omitempty"`
	ReduceOnly      bool     `json:"reduce_only,omitempty"`
	Slippage        *float64 `json:"slippage,omitempty"`
	OrderTag        string   `json:"order_tag,omitempty"`
	Level           *int     `json:"level,omitempty"`
	PostOnlyAdjust  *bool    `json:"post_only_adjust,omitempty"`
}

// CreateOrderResponse represents the response from POST /v1/order.
type CreateOrderResponse struct {
	OrderlyResponse
	Data CreateOrderData `json:"data"`
}

// CreateOrderData is the response data for create order endpoints.
type CreateOrderData struct {
	OrderID       StringOrNumber `json:"order_id"`
	ClientOrderID string         `json:"client_order_id"`
	OrderType     string         `json:"order_type"`
	OrderPrice    float64        `json:"order_price"`
	OrderQuantity float64        `json:"order_quantity"`
	AlgoType      string         `json:"algo_type,omitempty"`
	ErrorMessage  string         `json:"error_message"`
}

// CancelResponse represents the response from cancel order endpoints.
type CancelResponse struct {
	OrderlyResponse
	Data struct {
		Status string `json:"status"`
	} `json:"data"`
}

// BatchCreateOrderRequest represents the request body for POST /v1/batch-order.
type BatchCreateOrderRequest struct {
	Orders []CreateOrderRequest `json:"orders"`
}

// BatchCreateOrderResponse represents the response from POST /v1/batch-order.
type BatchCreateOrderResponse struct {
	OrderlyResponse
	Data struct {
		Rows []CreateOrderData `json:"rows"`
	} `json:"data"`
}

// CreateAlgoOrderRequest represents the request body for POST /v1/algo/order.
type CreateAlgoOrderRequest struct {
	Symbol           string                   `json:"symbol"`
	AlgoType         string                   `json:"algo_type"`
	Type             string                   `json:"type,omitempty"`
	Quantity         *float64                 `json:"quantity,omitempty"`
	Side             string                   `json:"side,omitempty"`
	ClientOrderID    string                   `json:"client_order_id,omitempty"`
	Price            *float64                 `json:"price,omitempty"`
	TriggerPriceType string                   `json:"trigger_price_type,omitempty"`
	TriggerPrice     *float64                 `json:"trigger_price,omitempty"`
	ReduceOnly       bool                     `json:"reduce_only,omitempty"`
	VisibleQuantity  *float64                 `json:"visible_quantity,omitempty"`
	OrderTag         string                   `json:"order_tag,omitempty"`
	ActivatedPrice   *float64                 `json:"activatedPrice,omitempty"`
	CallbackRate     string                   `json:"callbackRate,omitempty"`
	CallbackValue    string                   `json:"callbackValue,omitempty"`
	ChildOrders      []CreateAlgoOrderRequest `json:"child_orders,omitempty"`
}

// CreateAlgoOrderResponse represents the response from POST /v1/algo/order.
type CreateAlgoOrderResponse struct {
	OrderlyResponse
	Data struct {
		OrderID       StringOrNumber `json:"order_id"`
		ClientOrderID string         `json:"client_order_id"`
		AlgoType      string         `json:"algo_type"`
		Quantity      float64        `json:"quantity"`
	} `json:"data"`
}

// StringOrNumber unmarshals a JSON string or number and keeps the string form.
// This is useful for IDs that might be numeric on some exchanges but should be
// treated as strings across plugins.
type StringOrNumber string

func (s *StringOrNumber) UnmarshalJSON(b []byte) error {
	// Try string
	var str string
	if err := json.Unmarshal(b, &str); err == nil {
		*s = StringOrNumber(str)
		return nil
	}
	// Try number; json.Number preserves integer formatting.
	var num json.Number
	if err := json.Unmarshal(b, &num); err == nil {
		*s = StringOrNumber(num.String())
		return nil
	}
	// Try null
	if string(b) == "null" {
		*s = ""
		return nil
	}
	return &json.UnmarshalTypeError{Value: string(b), Type: nil}
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
