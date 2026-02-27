package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	ex "github.com/plusev-terminal/go-plugin-common/datasrc/exchange"
	acct "github.com/plusev-terminal/go-plugin-common/datasrc/exchange/account"
	"github.com/plusev-terminal/go-plugin-common/plugin"
	"github.com/trading-peter/plusev_datasource_orderly_plugin/orderly"
)

type indexedOrder struct {
	index int
	order ex.CreateOrderParams
}

func (p *OrderlyPlugin) handleCreateOrders(params map[string]any) plugin.Response {
	if p.client == nil {
		return plugin.ErrorResponseMsg("client not initialized")
	}

	parsed := ex.CreateOrdersParamsFromMap(params)
	if err := parsed.Validate(); err != nil {
		return plugin.ErrorResponse(err)
	}

	results := make([]acct.CreateOrderResult, len(parsed.Orders))

	for i := range results {
		results[i] = acct.CreateOrderResult{Index: i}
	}

	pending := make([]indexedOrder, 0, len(parsed.Orders))

	for i, order := range parsed.Orders {
		pending = append(pending, indexedOrder{index: i, order: order})
	}

	used := make(map[int]bool)

	groups := map[string][]indexedOrder{}

	for _, item := range pending {
		if item.order.GroupID == "" {
			continue
		}
		groups[item.order.GroupID] = append(groups[item.order.GroupID], item)
	}

	for groupID, items := range groups {
		if len(items) < 2 {
			continue
		}

		algoReq, indices, err := buildBracketAlgoOrder(groupID, items)
		if err != nil {
			for _, item := range items {
				results[item.index].Success = false
				results[item.index].Error = err.Error()
				results[item.index].GroupID = groupID
				used[item.index] = true
			}
			continue
		}

		resp, err := p.client.CreateAlgoOrder(algoReq)
		if err != nil {
			for _, idx := range indices {
				results[idx].Success = false
				results[idx].Error = err.Error()
				results[idx].GroupID = groupID
				used[idx] = true
			}
			continue
		}

		orderID := string(resp.Data.OrderID)
		clientOrderID := resp.Data.ClientOrderID

		for _, idx := range indices {
			results[idx].Success = true
			results[idx].ClientOrderID = parsed.Orders[idx].ClientOrderID
			results[idx].GroupID = groupID
			results[idx].Order = buildOrderResult(parsed.Orders[idx], orderID, clientOrderID)
			results[idx].Extra = map[string]any{"algoType": algoReq.AlgoType}
			used[idx] = true
		}
	}

	normalOrders := make([]indexedOrder, 0)
	algoOrders := make([]indexedOrder, 0)

	for _, item := range pending {
		if used[item.index] {
			continue
		}

		if isAlgoOrder(item.order) {
			algoOrders = append(algoOrders, item)
		} else {
			normalOrders = append(normalOrders, item)
		}
	}

	if len(normalOrders) > 1 {
		batchReq := orderly.BatchCreateOrderRequest{Orders: make([]orderly.CreateOrderRequest, 0, len(normalOrders))}

		for _, item := range normalOrders {
			req, err := buildCreateOrderRequest(item.order)
			if err != nil {
				results[item.index].Success = false
				results[item.index].Error = err.Error()
				continue
			}

			batchReq.Orders = append(batchReq.Orders, req)
		}

		if len(batchReq.Orders) > 0 {
			batchResp, err := p.client.BatchCreateOrder(batchReq)
			if err != nil {
				for _, item := range normalOrders {
					if results[item.index].Success || results[item.index].Error != "" {
						continue
					}
					results[item.index].Success = false
					results[item.index].Error = err.Error()
					results[item.index].Extra = map[string]any{"batch": true}
					used[item.index] = true
				}
			} else {
				for i, row := range batchResp.Data.Rows {
					if i >= len(normalOrders) {
						break
					}

					item := normalOrders[i]
					res := &results[item.index]
					errMsg := strings.TrimSpace(row.ErrorMessage)
					res.Success = errMsg == "" || strings.EqualFold(errMsg, "none")

					if !res.Success {
						res.Error = errMsg
					}

					res.ClientOrderID = row.ClientOrderID
					res.Order = buildOrderResult(item.order, string(row.OrderID), row.ClientOrderID)
					res.Extra = map[string]any{"batch": true}
					used[item.index] = true
				}
			}
		}
	}

	for _, item := range normalOrders {
		if used[item.index] {
			continue
		}

		req, err := buildCreateOrderRequest(item.order)
		if err != nil {
			results[item.index].Success = false
			results[item.index].Error = err.Error()
			continue
		}

		resp, err := p.client.CreateOrder(req)
		if err != nil {
			results[item.index].Success = false
			results[item.index].Error = err.Error()
			continue
		}

		errMsg := strings.TrimSpace(resp.Data.ErrorMessage)
		results[item.index].Success = errMsg == "" || strings.EqualFold(errMsg, "none")

		if !results[item.index].Success {
			results[item.index].Error = errMsg
		}

		results[item.index].ClientOrderID = resp.Data.ClientOrderID
		results[item.index].Order = buildOrderResult(item.order, string(resp.Data.OrderID), resp.Data.ClientOrderID)
		used[item.index] = true
	}

	for _, item := range algoOrders {
		if used[item.index] {
			continue
		}

		req, err := buildAlgoOrderRequest(item.order)
		if err != nil {
			results[item.index].Success = false
			results[item.index].Error = err.Error()
			continue
		}

		resp, err := p.client.CreateAlgoOrder(req)
		if err != nil {
			results[item.index].Success = false
			results[item.index].Error = err.Error()
			continue
		}

		results[item.index].Success = true
		results[item.index].ClientOrderID = resp.Data.ClientOrderID
		results[item.index].Order = buildOrderResult(item.order, string(resp.Data.OrderID), resp.Data.ClientOrderID)
		results[item.index].Extra = map[string]any{"algoType": req.AlgoType}
		used[item.index] = true
	}

	resp := acct.CreateOrdersResponse{
		SubmittedAt: time.Now(),
		Results:     results,
	}

	return plugin.SuccessResponse(resp)
}

func buildOrderResult(order ex.CreateOrderParams, orderID string, clientOrderID string) *acct.Order {
	result := &acct.Order{
		OrderID:       orderID,
		ClientOrderID: clientOrderID,
		Symbol:        getOrderSymbol(order),
		Side:          order.Side,
		Type:          order.Type,
		Price:         order.Price,
		Quantity:      order.Quantity,
		ReduceOnly:    order.ReduceOnly,
		PostOnly:      order.PostOnly,
	}

	return result
}

func isAlgoOrder(order ex.CreateOrderParams) bool {
	switch order.Type {
	case acct.OrderTypeStop,
		acct.OrderTypeStopLimit,
		acct.OrderTypeStopMarket,
		acct.OrderTypeTakeProfit,
		acct.OrderTypeTakeProfitLimit,
		acct.OrderTypeTakeProfitMarket,
		acct.OrderTypeTrailingStop:
		return true
	}

	if order.TriggerPrice != "" || order.StopPrice != "" || order.TakeProfitPrice != "" || order.TrailingDelta != "" || order.ActivationPrice != "" {
		return true
	}

	if order.ClosePosition {
		return true
	}

	return false
}

func buildCreateOrderRequest(order ex.CreateOrderParams) (orderly.CreateOrderRequest, error) {
	symbol := getOrderSymbol(order)

	if symbol == "" {
		return orderly.CreateOrderRequest{}, fmt.Errorf("symbol is required")
	}

	orderType, err := mapOrderlyOrderType(order)

	if err != nil {
		return orderly.CreateOrderRequest{}, err
	}

	req := orderly.CreateOrderRequest{
		Symbol:        symbol,
		OrderType:     orderType,
		Side:          orderlySide(order.Side),
		ClientOrderID: order.ClientOrderID,
		ReduceOnly:    order.ReduceOnly,
		OrderTag:      pickOrderTag(order),
	}

	if req.Side == "" {
		return orderly.CreateOrderRequest{}, fmt.Errorf("side is required")
	}

	if qty := strings.TrimSpace(order.Quantity); qty != "" {
		v, err := parseFloat(qty)

		if err != nil {
			return orderly.CreateOrderRequest{}, fmt.Errorf("invalid quantity: %w", err)
		}
		req.OrderQuantity = &v
	} else if amt := strings.TrimSpace(order.QuoteQuantity); amt != "" {
		v, err := parseFloat(amt)

		if err != nil {
			return orderly.CreateOrderRequest{}, fmt.Errorf("invalid quoteQuantity: %w", err)
		}

		req.OrderAmount = &v
	} else {
		return orderly.CreateOrderRequest{}, fmt.Errorf("quantity or quoteQuantity is required")
	}

	if requiresPrice(orderType) {
		if strings.TrimSpace(order.Price) == "" {
			return orderly.CreateOrderRequest{}, fmt.Errorf("price is required for %s order", orderType)
		}

		v, err := parseFloat(order.Price)
		if err != nil {
			return orderly.CreateOrderRequest{}, fmt.Errorf("invalid price: %w", err)
		}

		req.OrderPrice = &v
	}

	if v, ok := extraFloat(order.Extra, "visibleQuantity", "visible_quantity"); ok {
		req.VisibleQuantity = &v
	}

	if v, ok := extraFloat(order.Extra, "slippage"); ok {
		req.Slippage = &v
	}

	if v, ok := extraInt(order.Extra, "level"); ok {
		req.Level = &v
	}

	if v, ok := extraBool(order.Extra, "postOnlyAdjust", "post_only_adjust"); ok {
		req.PostOnlyAdjust = &v
	}

	return req, nil
}

func buildAlgoOrderRequest(order ex.CreateOrderParams) (orderly.CreateAlgoOrderRequest, error) {
	symbol := getOrderSymbol(order)
	if symbol == "" {
		return orderly.CreateAlgoOrderRequest{}, fmt.Errorf("symbol is required")
	}

	if isTrailingStop(order) {
		return buildTrailingStopOrder(symbol, order)
	}

	if hasBothTPSL(order) {
		return buildTpSlOrder(symbol, order)
	}

	if order.ClosePosition || order.ReduceOnly {
		if order.TakeProfitPrice != "" || order.StopPrice != "" {
			return buildPositionalTpSl(symbol, order)
		}
	}

	return buildStopAlgoOrder(symbol, order)
}

func buildStopAlgoOrder(symbol string, order ex.CreateOrderParams) (orderly.CreateAlgoOrderRequest, error) {
	req := orderly.CreateAlgoOrderRequest{
		Symbol:        symbol,
		AlgoType:      "STOP",
		ClientOrderID: order.ClientOrderID,
		ReduceOnly:    order.ReduceOnly,
		OrderTag:      pickOrderTag(order),
	}

	req.Side = orderlySide(order.Side)

	if req.Side == "" {
		return orderly.CreateAlgoOrderRequest{}, fmt.Errorf("side is required")
	}

	triggerPrice := pickTriggerPrice(order)

	if triggerPrice == "" {
		return orderly.CreateAlgoOrderRequest{}, fmt.Errorf("triggerPrice or stopPrice or takeProfitPrice is required")
	}

	triggerValue, err := parseFloat(triggerPrice)

	if err != nil {
		return orderly.CreateAlgoOrderRequest{}, fmt.Errorf("invalid trigger price: %w", err)
	}

	triggerType := pickTriggerPriceType(order)

	if triggerType != "" {
		req.TriggerPriceType = triggerType
	}

	req.TriggerPrice = &triggerValue

	if qty := strings.TrimSpace(order.Quantity); qty != "" {
		v, err := parseFloat(qty)
		if err != nil {
			return orderly.CreateAlgoOrderRequest{}, fmt.Errorf("invalid quantity: %w", err)
		}
		req.Quantity = &v
	}

	if requiresStopLimit(order) {
		req.Type = "LIMIT"
		if strings.TrimSpace(order.Price) == "" {
			return orderly.CreateAlgoOrderRequest{}, fmt.Errorf("price is required for stop limit order")
		}
		v, err := parseFloat(order.Price)
		if err != nil {
			return orderly.CreateAlgoOrderRequest{}, fmt.Errorf("invalid price: %w", err)
		}
		req.Price = &v
	} else {
		req.Type = "MARKET"
	}

	if v, ok := extraFloat(order.Extra, "visibleQuantity", "visible_quantity"); ok {
		req.VisibleQuantity = &v
	}

	return req, nil
}

func buildTpSlOrder(symbol string, order ex.CreateOrderParams) (orderly.CreateAlgoOrderRequest, error) {
	req := orderly.CreateAlgoOrderRequest{
		Symbol:           symbol,
		AlgoType:         "TP_SL",
		ClientOrderID:    order.ClientOrderID,
		OrderTag:         pickOrderTag(order),
		TriggerPriceType: "MARK_PRICE",
	}

	if qty := strings.TrimSpace(order.Quantity); qty != "" {
		v, err := parseFloat(qty)
		if err != nil {
			return orderly.CreateAlgoOrderRequest{}, fmt.Errorf("invalid quantity: %w", err)
		}

		req.Quantity = &v
	} else {
		return orderly.CreateAlgoOrderRequest{}, fmt.Errorf("quantity is required for TP_SL")
	}

	req.Side = orderlySide(order.Side)

	if req.Side == "" {
		return orderly.CreateAlgoOrderRequest{}, fmt.Errorf("side is required")
	}

	tpChild, err := buildTpChild(symbol, order, "MARKET")
	if err != nil {
		return orderly.CreateAlgoOrderRequest{}, err
	}

	slChild, err := buildSlChild(symbol, order, "MARKET")
	if err != nil {
		return orderly.CreateAlgoOrderRequest{}, err
	}

	req.ChildOrders = []orderly.CreateAlgoOrderRequest{tpChild, slChild}
	return req, nil
}

func buildPositionalTpSl(symbol string, order ex.CreateOrderParams) (orderly.CreateAlgoOrderRequest, error) {
	req := orderly.CreateAlgoOrderRequest{
		Symbol:           symbol,
		AlgoType:         "POSITIONAL_TP_SL",
		ClientOrderID:    order.ClientOrderID,
		OrderTag:         pickOrderTag(order),
		TriggerPriceType: "MARK_PRICE",
	}

	childOrders := make([]orderly.CreateAlgoOrderRequest, 0, 2)
	if strings.TrimSpace(order.TakeProfitPrice) != "" {
		tpChild, err := buildTpChild(symbol, order, "CLOSE_POSITION")
		if err != nil {
			return orderly.CreateAlgoOrderRequest{}, err
		}
		childOrders = append(childOrders, tpChild)
	}

	if strings.TrimSpace(order.StopPrice) != "" || strings.TrimSpace(order.TriggerPrice) != "" {
		slChild, err := buildSlChild(symbol, order, "CLOSE_POSITION")
		if err != nil {
			return orderly.CreateAlgoOrderRequest{}, err
		}
		childOrders = append(childOrders, slChild)
	}

	if len(childOrders) == 0 {
		return orderly.CreateAlgoOrderRequest{}, fmt.Errorf("takeProfitPrice or stopPrice is required for POSITIONAL_TP_SL")
	}

	req.ChildOrders = childOrders
	return req, nil
}

func buildTrailingStopOrder(symbol string, order ex.CreateOrderParams) (orderly.CreateAlgoOrderRequest, error) {
	req := orderly.CreateAlgoOrderRequest{
		Symbol:        symbol,
		AlgoType:      "TRAILING_STOP",
		Type:          "MARKET",
		ClientOrderID: order.ClientOrderID,
		ReduceOnly:    order.ReduceOnly,
		OrderTag:      pickOrderTag(order),
	}

	req.Side = orderlySide(order.Side)
	if req.Side == "" {
		return orderly.CreateAlgoOrderRequest{}, fmt.Errorf("side is required")
	}

	if qty := strings.TrimSpace(order.Quantity); qty != "" {
		v, err := parseFloat(qty)
		if err != nil {
			return orderly.CreateAlgoOrderRequest{}, fmt.Errorf("invalid quantity: %w", err)
		}
		req.Quantity = &v
	}

	if v, ok := extraString(order.Extra, "callbackRate"); ok {
		req.CallbackRate = v
	}

	if v, ok := extraString(order.Extra, "callbackValue"); ok {
		req.CallbackValue = v
	}

	if req.CallbackRate == "" && req.CallbackValue == "" && strings.TrimSpace(order.TrailingDelta) != "" {
		req.CallbackValue = strings.TrimSpace(order.TrailingDelta)
	}

	if strings.TrimSpace(order.ActivationPrice) != "" {
		v, err := parseFloat(order.ActivationPrice)
		if err != nil {
			return orderly.CreateAlgoOrderRequest{}, fmt.Errorf("invalid activationPrice: %w", err)
		}
		req.ActivatedPrice = &v
	}

	return req, nil
}

func buildBracketAlgoOrder(groupID string, items []indexedOrder) (orderly.CreateAlgoOrderRequest, []int, error) {
	var entry *indexedOrder
	children := make([]indexedOrder, 0)

	for i := range items {
		order := items[i].order
		if entry == nil && !order.ReduceOnly && !order.ClosePosition && !isAlgoOrder(order) {
			entry = &items[i]
			continue
		}
		children = append(children, items[i])
	}

	if entry == nil {
		return orderly.CreateAlgoOrderRequest{}, nil, fmt.Errorf("no entry order for bracket")
	}

	symbol := getOrderSymbol(entry.order)

	if symbol == "" {
		return orderly.CreateAlgoOrderRequest{}, nil, fmt.Errorf("symbol is required")
	}

	qty := strings.TrimSpace(entry.order.Quantity)

	if qty == "" {
		return orderly.CreateAlgoOrderRequest{}, nil, fmt.Errorf("quantity is required for bracket entry")
	}

	qtyValue, err := parseFloat(qty)

	if err != nil {
		return orderly.CreateAlgoOrderRequest{}, nil, fmt.Errorf("invalid entry quantity: %w", err)
	}

	entryType := "MARKET"
	if entry.order.Type == acct.OrderTypeLimit {
		entryType = "LIMIT"
	}

	var price *float64

	if entryType == "LIMIT" {
		if strings.TrimSpace(entry.order.Price) == "" {
			return orderly.CreateAlgoOrderRequest{}, nil, fmt.Errorf("price is required for bracket entry limit")
		}

		v, err := parseFloat(entry.order.Price)
		if err != nil {
			return orderly.CreateAlgoOrderRequest{}, nil, fmt.Errorf("invalid entry price: %w", err)
		}

		price = &v
	}

	entrySide := orderlySide(entry.order.Side)
	if entrySide == "" {
		return orderly.CreateAlgoOrderRequest{}, nil, fmt.Errorf("side is required for bracket entry")
	}

	tpChild, slChild := pickBracketChildren(symbol, entry.order, children)
	if tpChild == nil && slChild == nil {
		return orderly.CreateAlgoOrderRequest{}, nil, fmt.Errorf("no TP/SL child orders for bracket")
	}

	positional := orderly.CreateAlgoOrderRequest{
		Symbol:           symbol,
		AlgoType:         "POSITIONAL_TP_SL",
		TriggerPriceType: "MARK_PRICE",
		ChildOrders:      []orderly.CreateAlgoOrderRequest{},
	}

	if tpChild != nil {
		positional.ChildOrders = append(positional.ChildOrders, *tpChild)
	}

	if slChild != nil {
		positional.ChildOrders = append(positional.ChildOrders, *slChild)
	}

	req := orderly.CreateAlgoOrderRequest{
		Symbol:        symbol,
		AlgoType:      "BRACKET",
		Type:          entryType,
		Quantity:      &qtyValue,
		Side:          entrySide,
		ClientOrderID: entry.order.ClientOrderID,
		OrderTag:      sanitizeOrderTag(groupID),
		ChildOrders:   []orderly.CreateAlgoOrderRequest{positional},
	}

	if price != nil {
		req.Price = price
	}

	indices := make([]int, 0, len(children)+1)
	indices = append(indices, entry.index)

	for _, child := range children {
		indices = append(indices, child.index)
	}

	return req, indices, nil
}

func pickBracketChildren(symbol string, entry ex.CreateOrderParams, children []indexedOrder) (*orderly.CreateAlgoOrderRequest, *orderly.CreateAlgoOrderRequest) {
	var tpChild *orderly.CreateAlgoOrderRequest
	var slChild *orderly.CreateAlgoOrderRequest

	for _, child := range children {
		order := child.order
		if order.TakeProfitPrice != "" || order.Type == acct.OrderTypeTakeProfit || order.Type == acct.OrderTypeTakeProfitMarket || order.Type == acct.OrderTypeTakeProfitLimit {
			c, err := buildTpChild(symbol, order, "CLOSE_POSITION")
			if err == nil {
				tpChild = &c
			}
			continue
		}

		if order.StopPrice != "" || order.TriggerPrice != "" || order.Type == acct.OrderTypeStop || order.Type == acct.OrderTypeStopMarket || order.Type == acct.OrderTypeStopLimit {
			c, err := buildSlChild(symbol, order, "CLOSE_POSITION")
			if err == nil {
				slChild = &c
			}
		}
	}

	if tpChild != nil {
		if tpChild.Side == "" {
			tpChild.Side = oppositeOrderlySide(entry.Side)
		}
	}

	if slChild != nil {
		if slChild.Side == "" {
			slChild.Side = oppositeOrderlySide(entry.Side)
		}
	}

	return tpChild, slChild
}

func buildTpChild(symbol string, order ex.CreateOrderParams, childType string) (orderly.CreateAlgoOrderRequest, error) {
	tp := strings.TrimSpace(order.TakeProfitPrice)
	if tp == "" {
		return orderly.CreateAlgoOrderRequest{}, fmt.Errorf("takeProfitPrice is required")
	}

	price, err := parseFloat(tp)
	if err != nil {
		return orderly.CreateAlgoOrderRequest{}, fmt.Errorf("invalid takeProfitPrice: %w", err)
	}

	child := orderly.CreateAlgoOrderRequest{
		Symbol:       symbol,
		AlgoType:     "TAKE_PROFIT",
		Type:         childType,
		Side:         orderlySide(order.Side),
		TriggerPrice: &price,
		ReduceOnly:   true,
	}

	return child, nil
}

func buildSlChild(symbol string, order ex.CreateOrderParams, childType string) (orderly.CreateAlgoOrderRequest, error) {
	sl := strings.TrimSpace(order.StopPrice)
	if sl == "" {
		sl = strings.TrimSpace(order.TriggerPrice)
	}

	if sl == "" {
		return orderly.CreateAlgoOrderRequest{}, fmt.Errorf("stopPrice is required")
	}

	price, err := parseFloat(sl)
	if err != nil {
		return orderly.CreateAlgoOrderRequest{}, fmt.Errorf("invalid stopPrice: %w", err)
	}

	child := orderly.CreateAlgoOrderRequest{
		Symbol:       symbol,
		AlgoType:     "STOP_LOSS",
		Type:         childType,
		Side:         orderlySide(order.Side),
		TriggerPrice: &price,
		ReduceOnly:   true,
	}

	return child, nil
}

func mapOrderlyOrderType(order ex.CreateOrderParams) (string, error) {
	switch order.Type {
	case acct.OrderTypeLimit:
		if order.PostOnly || order.TimeInForce == acct.TimeInForcePostOnly || order.TimeInForce == acct.TimeInForceGTX {
			return "POST_ONLY", nil
		}

		if order.TimeInForce == acct.TimeInForceIOC {
			return "IOC", nil
		}

		if order.TimeInForce == acct.TimeInForceFOK {
			return "FOK", nil
		}

		return "LIMIT", nil
	case acct.OrderTypeMarket:
		return "MARKET", nil
	default:
		return "", fmt.Errorf("unsupported order type for create order: %s", order.Type)
	}
}

func requiresPrice(orderType string) bool {
	switch orderType {
	case "LIMIT", "IOC", "FOK", "POST_ONLY":
		return true
	default:
		return false
	}
}

func requiresStopLimit(order ex.CreateOrderParams) bool {
	return order.Type == acct.OrderTypeStopLimit
}

func getOrderSymbol(order ex.CreateOrderParams) string {
	return strings.TrimSpace(order.Market.Symbol)
}

func orderlySide(side acct.OrderSide) string {
	s := strings.ToLower(strings.TrimSpace(string(side)))
	switch s {
	case "buy":
		return "BUY"
	case "sell":
		return "SELL"
	default:
		return ""
	}
}

func oppositeOrderlySide(side acct.OrderSide) string {
	s := strings.ToLower(strings.TrimSpace(string(side)))
	if s == "buy" {
		return "SELL"
	}

	if s == "sell" {
		return "BUY"
	}

	return ""
}

func parseFloat(value string) (float64, error) {
	v, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0, err
	}

	return v, nil
}

func pickOrderTag(order ex.CreateOrderParams) string {
	if v, ok := extraString(order.Extra, "orderTag"); ok {
		return sanitizeOrderTag(v)
	}
	return sanitizeOrderTag(order.GroupID)
}

func sanitizeOrderTag(tag string) string {
	trimmed := strings.TrimSpace(tag)
	if trimmed == "" {
		return ""
	}

	return strings.Map(func(r rune) rune {
		if r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r) {
			return r
		}
		return '_'
	}, trimmed)
}

func pickTriggerPrice(order ex.CreateOrderParams) string {
	if strings.TrimSpace(order.TriggerPrice) != "" {
		return order.TriggerPrice
	}

	if strings.TrimSpace(order.StopPrice) != "" {
		return order.StopPrice
	}

	if strings.TrimSpace(order.TakeProfitPrice) != "" {
		return order.TakeProfitPrice
	}

	return ""
}

func pickTriggerPriceType(order ex.CreateOrderParams) string {
	if v, ok := extraString(order.Extra, "triggerPriceType", "trigger_price_type"); ok {
		return strings.ToUpper(strings.TrimSpace(v))
	}
	return "MARK_PRICE"
}

func hasBothTPSL(order ex.CreateOrderParams) bool {
	return strings.TrimSpace(order.TakeProfitPrice) != "" && strings.TrimSpace(order.StopPrice) != ""
}

func isTrailingStop(order ex.CreateOrderParams) bool {
	if order.Type == acct.OrderTypeTrailingStop {
		return true
	}

	if strings.TrimSpace(order.TrailingDelta) != "" || strings.TrimSpace(order.ActivationPrice) != "" {
		return true
	}

	if _, ok := extraString(order.Extra, "callbackRate", "callbackValue"); ok {
		return true
	}

	return false
}

func extraString(extra map[string]any, keys ...string) (string, bool) {
	for _, key := range keys {
		if v, ok := extra[key]; ok {
			switch t := v.(type) {
			case string:
				if strings.TrimSpace(t) != "" {
					return strings.TrimSpace(t), true
				}
			case fmt.Stringer:
				return strings.TrimSpace(t.String()), true
			}
		}
	}

	return "", false
}

func extraFloat(extra map[string]any, keys ...string) (float64, bool) {
	for _, key := range keys {
		if v, ok := extra[key]; ok {
			switch t := v.(type) {
			case float64:
				return t, true
			case int:
				return float64(t), true
			case int64:
				return float64(t), true
			case string:
				if strings.TrimSpace(t) == "" {
					continue
				}

				parsed, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
				if err == nil {
					return parsed, true
				}
			}
		}
	}
	return 0, false
}

func extraInt(extra map[string]any, keys ...string) (int, bool) {
	for _, key := range keys {
		if v, ok := extra[key]; ok {
			switch t := v.(type) {
			case int:
				return t, true
			case int64:
				return int(t), true
			case float64:
				return int(t), true
			case string:
				if strings.TrimSpace(t) == "" {
					continue
				}

				parsed, err := strconv.Atoi(strings.TrimSpace(t))
				if err == nil {
					return parsed, true
				}
			}
		}
	}
	return 0, false
}

func extraBool(extra map[string]any, keys ...string) (bool, bool) {
	for _, key := range keys {
		if v, ok := extra[key]; ok {
			switch t := v.(type) {
			case bool:
				return t, true
			case string:
				parsed := strings.TrimSpace(strings.ToLower(t))
				if parsed == "true" || parsed == "1" {
					return true, true
				}

				if parsed == "false" || parsed == "0" {
					return false, true
				}
			case float64:
				return t != 0, true
			}
		}
	}
	return false, false
}
