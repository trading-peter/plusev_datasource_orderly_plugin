package main

import (
	"fmt"
	"strings"
	"time"

	ex "github.com/plusev-terminal/go-plugin-common/datasrc/exchange"
	acct "github.com/plusev-terminal/go-plugin-common/datasrc/exchange/account"
	"github.com/plusev-terminal/go-plugin-common/plugin"
	"github.com/trading-peter/plusev_datasource_orderly_plugin/orderly"
)

func normalizeOrderlySide(side string) acct.OrderSide {
	s := strings.ToUpper(strings.TrimSpace(side))
	switch s {
	case "BUY", "BID", "LONG":
		return acct.OrderSideBuy
	case "SELL", "ASK", "SHORT":
		return acct.OrderSideSell
	case "":
		return acct.OrderSideUnknown
	default:
		return acct.OrderSide(strings.ToLower(s))
	}
}

func normalizeOrderlyType(orderType string) acct.OrderType {
	s := strings.ToUpper(strings.TrimSpace(orderType))
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")
	switch s {
	case "LIMIT":
		return acct.OrderTypeLimit
	case "MARKET":
		return acct.OrderTypeMarket
	case "STOP", "STOP_LOSS":
		return acct.OrderTypeStop
	case "STOP_LIMIT":
		return acct.OrderTypeStopLimit
	case "STOP_MARKET":
		return acct.OrderTypeStopMarket
	case "TAKE_PROFIT", "TP":
		return acct.OrderTypeTakeProfit
	case "TAKE_PROFIT_LIMIT":
		return acct.OrderTypeTakeProfitLimit
	case "TAKE_PROFIT_MARKET":
		return acct.OrderTypeTakeProfitMarket
	case "TRAILING_STOP":
		return acct.OrderTypeTrailingStop
	case "BRACKET":
		return acct.OrderTypeUnknown
	case "POSITIONAL_TPSL":
		return acct.OrderTypeUnknown
	case "":
		return acct.OrderTypeUnknown
	default:
		return acct.OrderType(strings.ToLower(s))
	}
}

func isOrderlyTPSL(orderType string) bool {
	s := strings.ToUpper(strings.TrimSpace(orderType))
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")
	return s == "TP_SL" || s == "TPSL"
}

func normalizeOrderlyStatus(status string) acct.OrderStatus {
	s := strings.ToUpper(strings.TrimSpace(status))
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")
	switch s {
	case "NEW":
		return acct.OrderStatusNew
	case "PARTIAL_FILLED", "PARTIALLY_FILLED":
		return acct.OrderStatusPartiallyFilled
	case "FILLED":
		return acct.OrderStatusFilled
	case "CANCELED", "CANCELLED":
		return acct.OrderStatusCanceled
	case "REJECTED":
		return acct.OrderStatusRejected
	case "EXPIRED":
		return acct.OrderStatusExpired
	case "INCOMPLETE":
		return acct.OrderStatusIncomplete
	case "COMPLETED":
		return acct.OrderStatusCompleted
	case "":
		return acct.OrderStatusUnknown
	default:
		return acct.OrderStatus(strings.ToLower(s))
	}
}

func (p *OrderlyPlugin) handleGetOrders(params map[string]any) plugin.Response {
	if p.client == nil {
		return plugin.ErrorResponseMsg("client not initialized")
	}

	parsed := ex.GetOrdersParamsFromMap(params)
	openOnly := parsed.OpenOnly

	// Fetch both normal orders and algo orders (stop-loss, TP/SL, etc)
	ordersResp, err := p.client.GetOrders(openOnly)
	if err != nil {
		return plugin.ErrorResponse(err)
	}

	algoOrdersResp, algoErr := p.client.GetAlgoOrders(openOnly)
	if algoErr != nil {
		// Log but don't fail; some accounts may not have algo orders enabled
		// or the endpoint may fail independently
		log.ErrorWithData("Failed to fetch algo orders", map[string]any{"error": algoErr})
	}

	fetchedAt := time.Now()
	if ordersResp.Timestamp > 0 {
		fetchedAt = time.UnixMilli(ordersResp.Timestamp)
	}

	if algoOrdersResp.Timestamp > 0 && algoOrdersResp.Timestamp > ordersResp.Timestamp {
		fetchedAt = time.UnixMilli(algoOrdersResp.Timestamp)
	}

	// Merge normal + algo orders into a single response list
	orders := make([]acct.Order, 0, len(ordersResp.Data.Rows)+len(algoOrdersResp.Data.Rows))

	// Add normal orders
	for _, o := range ordersResp.Data.Rows {
		remaining := o.Quantity - o.ExecutedQty
		orderType := normalizeOrderlyType(o.OrderType)
		orderStatus := normalizeOrderlyStatus(o.Status)

		order := acct.Order{
			OrderID:           string(o.OrderID),
			ClientOrderID:     o.ClientOrderID,
			Symbol:            o.Symbol,
			Side:              normalizeOrderlySide(o.Side),
			Type:              orderType,
			Status:            orderStatus,
			Price:             fmt.Sprintf("%.15g", o.Price),
			Quantity:          fmt.Sprintf("%.15g", o.Quantity),
			ExecutedQuantity:  fmt.Sprintf("%.15g", o.ExecutedQty),
			RemainingQuantity: fmt.Sprintf("%.15g", remaining),
			Components:        map[string]string{},
			Extra: map[string]any{
				"order_category": "normal",
			},
		}
		if o.CreatedTime > 0 {
			order.CreatedAt = time.UnixMilli(o.CreatedTime)
		}
		if o.UpdatedTime > 0 {
			order.UpdatedAt = time.UnixMilli(o.UpdatedTime)
		}

		orders = append(orders, order)
	}

	// Add algo orders
	var appendAlgoOrder func(a orderly.AlgoOrderRow, category string)

	appendAlgoOrder = func(a orderly.AlgoOrderRow, category string) {
		if !a.IsActivated {
			return
		}

		executed := a.ExecutedQuantity

		if executed == 0 {
			executed = a.TotalExecutedQuantity
		}

		remaining := a.Quantity - executed
		algoType := normalizeOrderlyType(a.AlgoType)
		baseType := normalizeOrderlyType(a.Type)
		orderType := baseType

		if algoType != acct.OrderTypeUnknown {
			orderType = algoType
		}

		isTPSL := isOrderlyTPSL(a.AlgoType) || isOrderlyTPSL(a.Type)
		hasChildren := len(a.ChildOrders) > 0

		status := a.AlgoStatus

		if strings.TrimSpace(status) == "" {
			status = a.Status
		}

		if !(isTPSL && hasChildren) {
			order := acct.Order{
				OrderID:           string(a.AlgoOrderID),
				Symbol:            a.Symbol,
				Side:              normalizeOrderlySide(a.Side),
				Type:              orderType,
				Status:            normalizeOrderlyStatus(status),
				Price:             fmt.Sprintf("%.15g", a.TriggerPrice),
				Quantity:          fmt.Sprintf("%.15g", a.Quantity),
				ExecutedQuantity:  fmt.Sprintf("%.15g", executed),
				RemainingQuantity: fmt.Sprintf("%.15g", remaining),
				ReduceOnly:        a.ReduceOnly,
				Components: map[string]string{
					"trigger_price":           fmt.Sprintf("%.15g", a.TriggerPrice),
					"trigger_status":          a.TriggerStatus,
					"trigger_price_type":      a.TriggerPriceType,
					"trigger_time":            fmt.Sprintf("%d", a.TriggerTime),
					"average_executed_price":  fmt.Sprintf("%.15g", a.AverageExecutedPrice),
					"total_fee":               fmt.Sprintf("%.15g", a.TotalFee),
					"realized_pnl":            fmt.Sprintf("%.15g", a.RealizedPnl),
					"total_executed_quantity": fmt.Sprintf("%.15g", a.TotalExecutedQuantity),
					"visible_quantity":        fmt.Sprintf("%.15g", a.VisibleQuantity),
					"base_type":               string(baseType),
				},
				Extra: map[string]any{
					"order_category":         category,
					"algo_type":              string(algoType),
					"is_triggered":           a.IsTriggered,
					"is_activated":           a.IsActivated,
					"root_algo_order_id":     string(a.RootAlgoOrderID),
					"parent_algo_order_id":   string(a.ParentAlgoOrderID),
					"root_algo_order_status": a.RootAlgoOrderStatus,
					"order_tag":              a.OrderTag,
					"fee_asset":              a.FeeAsset,
				},
			}

			if a.CreatedTime > 0 {
				order.CreatedAt = time.UnixMilli(a.CreatedTime)
			}

			if a.UpdatedTime > 0 {
				order.UpdatedAt = time.UnixMilli(a.UpdatedTime)
			}

			orders = append(orders, order)
		}

		for _, child := range a.ChildOrders {
			appendAlgoOrder(child, "algo_child")
		}
	}

	for _, a := range algoOrdersResp.Data.Rows {
		appendAlgoOrder(a, "algo")
	}

	respExtra := map[string]any{}
	if algoErr != nil {
		respExtra["algo_error"] = algoErr.Error()
	}

	resp := acct.OrdersResponse{
		FetchedAt: fetchedAt,
		Scopes: map[acct.ScopeType]acct.OrderScope{
			acct.ScopeFutures: {
				Orders: orders,
				Extra:  map[string]any{},
			},
		},
		Extra: respExtra,
	}

	// Small cache like other account-ish endpoints.
	return plugin.SuccessResponse(resp, time.Second*3)
}

// Ensure we reference ex package so go mod keeps it; also helps grep discoverability.
var _ = ex.CMD_GET_MARKETS
