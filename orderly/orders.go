package orderly

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	rt "github.com/plusev-terminal/go-plugin-common/requester/types"
)

// GetOrders fetches orders for the configured account.
// If openOnly is true, it requests only open orders (unfilled/partially filled).
func (c *Client) GetOrders(openOnly bool) (OrdersResponse, error) {
	u, err := url.Parse(c.baseURL + "/v1/orders")
	if err != nil {
		return OrdersResponse{}, fmt.Errorf("failed to parse orders url: %w", err)
	}

	q := u.Query()
	// Orderly API uses bundled statuses:
	// INCOMPLETE = NEW + PARTIAL_FILLED (open orders)
	// COMPLETED  = CANCELLED + FILLED
	// There is no documented `open_only` query parameter for GET /v1/orders.
	if openOnly {
		q.Set("status", "INCOMPLETE")
	}

	u.RawQuery = q.Encode()

	req := &rt.Request{
		Method: "GET",
		URL:    u.String(),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	c.addAuthHeaders(req, "")

	var resp OrdersResponse
	_, sendErr := c.requester.Send(req, &resp)
	if sendErr != nil {
		return OrdersResponse{}, fmt.Errorf("failed to fetch /v1/orders code=%d message=%s: %w", resp.Code, resp.Message, sendErr)
	}

	if !resp.Success {
		return OrdersResponse{}, fmt.Errorf("orderly /v1/orders returned success=false code=%d message=%s", resp.Code, resp.Message)
	}

	return resp, nil
}

// GetAlgoOrders fetches algo orders (STOP, TPSL, etc) for the configured account.
// If openOnly is true, it requests only open algo orders using status=INCOMPLETE.
func (c *Client) GetAlgoOrders(openOnly bool) (AlgoOrdersResponse, error) {
	u, err := url.Parse(c.baseURL + "/v1/algo/orders")
	if err != nil {
		return AlgoOrdersResponse{}, fmt.Errorf("failed to parse algo orders url: %w", err)
	}

	q := u.Query()
	if openOnly {
		q.Set("status", "INCOMPLETE")
	}

	u.RawQuery = q.Encode()

	req := &rt.Request{
		Method: "GET",
		URL:    u.String(),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	c.addAuthHeaders(req, "")

	var resp AlgoOrdersResponse
	_, sendErr := c.requester.Send(req, &resp)
	if sendErr != nil {
		return AlgoOrdersResponse{}, fmt.Errorf("failed to fetch /v1/algo/orders code=%d message=%s: %w", resp.Code, resp.Message, sendErr)
	}
	if !resp.Success {
		return AlgoOrdersResponse{}, fmt.Errorf("orderly /v1/algo/orders returned success=false code=%d message=%s", resp.Code, resp.Message)
	}

	return resp, nil
}

// CreateOrder places a single order using POST /v1/order.
func (c *Client) CreateOrder(order CreateOrderRequest) (CreateOrderResponse, error) {
	body, err := json.Marshal(order)
	if err != nil {
		return CreateOrderResponse{}, fmt.Errorf("failed to marshal create order: %w", err)
	}

	req := &rt.Request{
		Method: "POST",
		URL:    c.baseURL + "/v1/order",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: body,
	}

	c.addAuthHeaders(req, string(body))

	var resp CreateOrderResponse
	_, sendErr := c.requester.Send(req, &resp)
	if sendErr != nil {
		return CreateOrderResponse{}, fmt.Errorf("failed to create /v1/order code=%d message=%s: %w", resp.Code, resp.Message, sendErr)
	}
	if !resp.Success {
		return resp, fmt.Errorf("orderly /v1/order returned success=false code=%d message=%s", resp.Code, resp.Message)
	}

	return resp, nil
}

// BatchCreateOrder places multiple orders using POST /v1/batch-order.
func (c *Client) BatchCreateOrder(batch BatchCreateOrderRequest) (BatchCreateOrderResponse, error) {
	body, err := json.Marshal(batch)
	if err != nil {
		return BatchCreateOrderResponse{}, fmt.Errorf("failed to marshal batch create order: %w", err)
	}

	req := &rt.Request{
		Method: "POST",
		URL:    c.baseURL + "/v1/batch-order",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: body,
	}

	c.addAuthHeaders(req, string(body))

	var resp BatchCreateOrderResponse
	_, sendErr := c.requester.Send(req, &resp)
	if sendErr != nil {
		return BatchCreateOrderResponse{}, fmt.Errorf("failed to create /v1/batch-order code=%d message=%s: %w", resp.Code, resp.Message, sendErr)
	}
	if !resp.Success {
		return resp, fmt.Errorf("orderly /v1/batch-order returned success=false code=%d message=%s", resp.Code, resp.Message)
	}

	return resp, nil
}

// CreateAlgoOrder places an algo order using POST /v1/algo/order.
func (c *Client) CreateAlgoOrder(order CreateAlgoOrderRequest) (CreateAlgoOrderResponse, error) {
	body, err := json.Marshal(order)
	if err != nil {
		return CreateAlgoOrderResponse{}, fmt.Errorf("failed to marshal create algo order: %w", err)
	}

	req := &rt.Request{
		Method: "POST",
		URL:    c.baseURL + "/v1/algo/order",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: body,
	}

	c.addAuthHeaders(req, string(body))

	var resp CreateAlgoOrderResponse
	_, sendErr := c.requester.Send(req, &resp)
	if sendErr != nil {
		return CreateAlgoOrderResponse{}, fmt.Errorf("failed to create /v1/algo/order code=%d message=%s: %w", resp.Code, resp.Message, sendErr)
	}
	if !resp.Success {
		return resp, fmt.Errorf("orderly /v1/algo/order returned success=false code=%d message=%s", resp.Code, resp.Message)
	}

	return resp, nil
}

// CancelOrder cancels a single order by order_id.
func (c *Client) CancelOrder(orderID string, symbol string) (CancelResponse, error) {
	u, err := url.Parse(c.baseURL + "/v1/order")
	if err != nil {
		return CancelResponse{}, fmt.Errorf("failed to parse cancel order url: %w", err)
	}

	q := u.Query()
	q.Set("order_id", orderID)
	q.Set("symbol", symbol)
	u.RawQuery = q.Encode()

	req := &rt.Request{
		Method: "DELETE",
		URL:    u.String(),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	c.addAuthHeaders(req, "")

	var resp CancelResponse
	_, sendErr := c.requester.Send(req, &resp)
	if sendErr != nil {
		return CancelResponse{}, fmt.Errorf("failed to cancel /v1/order code=%d message=%s: %w", resp.Code, resp.Message, sendErr)
	}
	if !resp.Success {
		return resp, fmt.Errorf("orderly /v1/order returned success=false code=%d message=%s", resp.Code, resp.Message)
	}

	return resp, nil
}

// CancelOrderByClientID cancels a single order by client_order_id.
func (c *Client) CancelOrderByClientID(clientOrderID string, symbol string) (CancelResponse, error) {
	u, err := url.Parse(c.baseURL + "/v1/client/order")
	if err != nil {
		return CancelResponse{}, fmt.Errorf("failed to parse cancel client order url: %w", err)
	}

	q := u.Query()
	q.Set("client_order_id", clientOrderID)
	q.Set("symbol", symbol)
	u.RawQuery = q.Encode()

	req := &rt.Request{
		Method: "DELETE",
		URL:    u.String(),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	c.addAuthHeaders(req, "")

	var resp CancelResponse
	_, sendErr := c.requester.Send(req, &resp)
	if sendErr != nil {
		return CancelResponse{}, fmt.Errorf("failed to cancel /v1/client/order code=%d message=%s: %w", resp.Code, resp.Message, sendErr)
	}
	if !resp.Success {
		return resp, fmt.Errorf("orderly /v1/client/order returned success=false code=%d message=%s", resp.Code, resp.Message)
	}

	return resp, nil
}

// CancelAlgoOrder cancels a single algo order by order_id.
func (c *Client) CancelAlgoOrder(orderID string, symbol string) (CancelResponse, error) {
	u, err := url.Parse(c.baseURL + "/v1/algo/order")
	if err != nil {
		return CancelResponse{}, fmt.Errorf("failed to parse cancel algo order url: %w", err)
	}

	q := u.Query()
	q.Set("order_id", orderID)
	q.Set("symbol", symbol)
	u.RawQuery = q.Encode()

	req := &rt.Request{
		Method: "DELETE",
		URL:    u.String(),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	c.addAuthHeaders(req, "")

	var resp CancelResponse
	_, sendErr := c.requester.Send(req, &resp)
	if sendErr != nil {
		return CancelResponse{}, fmt.Errorf("failed to cancel /v1/algo/order code=%d message=%s: %w", resp.Code, resp.Message, sendErr)
	}
	if !resp.Success {
		return resp, fmt.Errorf("orderly /v1/algo/order returned success=false code=%d message=%s", resp.Code, resp.Message)
	}

	return resp, nil
}

// CancelAlgoOrderByClientID cancels a single algo order by client_order_id.
func (c *Client) CancelAlgoOrderByClientID(clientOrderID string, symbol string) (CancelResponse, error) {
	u, err := url.Parse(c.baseURL + "/v1/algo/client/order")
	if err != nil {
		return CancelResponse{}, fmt.Errorf("failed to parse cancel algo client order url: %w", err)
	}

	q := u.Query()
	q.Set("client_order_id", clientOrderID)
	q.Set("symbol", symbol)
	u.RawQuery = q.Encode()

	req := &rt.Request{
		Method: "DELETE",
		URL:    u.String(),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	c.addAuthHeaders(req, "")

	var resp CancelResponse
	_, sendErr := c.requester.Send(req, &resp)
	if sendErr != nil {
		return CancelResponse{}, fmt.Errorf("failed to cancel /v1/algo/client/order code=%d message=%s: %w", resp.Code, resp.Message, sendErr)
	}
	if !resp.Success {
		return resp, fmt.Errorf("orderly /v1/algo/client/order returned success=false code=%d message=%s", resp.Code, resp.Message)
	}

	return resp, nil
}

// CancelAllPendingOrders cancels all open orders, optionally filtered by symbol.
func (c *Client) CancelAllPendingOrders(symbol string) (CancelResponse, error) {
	u, err := url.Parse(c.baseURL + "/v1/orders")
	if err != nil {
		return CancelResponse{}, fmt.Errorf("failed to parse cancel all orders url: %w", err)
	}

	if symbol != "" {
		q := u.Query()
		q.Set("symbol", symbol)
		u.RawQuery = q.Encode()
	}

	req := &rt.Request{
		Method: "DELETE",
		URL:    u.String(),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	c.addAuthHeaders(req, "")

	var resp CancelResponse
	_, sendErr := c.requester.Send(req, &resp)
	if sendErr != nil {
		return CancelResponse{}, fmt.Errorf("failed to cancel /v1/orders code=%d message=%s: %w", resp.Code, resp.Message, sendErr)
	}
	if !resp.Success {
		return resp, fmt.Errorf("orderly /v1/orders returned success=false code=%d message=%s", resp.Code, resp.Message)
	}

	return resp, nil
}

// CancelAllPendingAlgoOrders cancels all open algo orders, optionally filtered by symbol.
func (c *Client) CancelAllPendingAlgoOrders(symbol string) (CancelResponse, error) {
	u, err := url.Parse(c.baseURL + "/v1/algo/orders")
	if err != nil {
		return CancelResponse{}, fmt.Errorf("failed to parse cancel all algo orders url: %w", err)
	}

	if symbol != "" {
		q := u.Query()
		q.Set("symbol", symbol)
		u.RawQuery = q.Encode()
	}

	req := &rt.Request{
		Method: "DELETE",
		URL:    u.String(),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	c.addAuthHeaders(req, "")

	var resp CancelResponse
	_, sendErr := c.requester.Send(req, &resp)
	if sendErr != nil {
		return CancelResponse{}, fmt.Errorf("failed to cancel /v1/algo/orders code=%d message=%s: %w", resp.Code, resp.Message, sendErr)
	}
	if !resp.Success {
		return resp, fmt.Errorf("orderly /v1/algo/orders returned success=false code=%d message=%s", resp.Code, resp.Message)
	}

	return resp, nil
}

// BatchCancelOrders cancels a list of orders by order_ids (max 10 per request).
func (c *Client) BatchCancelOrders(orderIDs []string) (CancelResponse, error) {
	u, err := url.Parse(c.baseURL + "/v1/batch-order")
	if err != nil {
		return CancelResponse{}, fmt.Errorf("failed to parse batch cancel order url: %w", err)
	}

	q := u.Query()
	q.Set("order_ids", strings.Join(orderIDs, ","))
	u.RawQuery = q.Encode()

	req := &rt.Request{
		Method: "DELETE",
		URL:    u.String(),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	c.addAuthHeaders(req, "")

	var resp CancelResponse
	_, sendErr := c.requester.Send(req, &resp)
	if sendErr != nil {
		return CancelResponse{}, fmt.Errorf("failed to cancel /v1/batch-order code=%d message=%s: %w", resp.Code, resp.Message, sendErr)
	}
	if !resp.Success {
		return resp, fmt.Errorf("orderly /v1/batch-order returned success=false code=%d message=%s", resp.Code, resp.Message)
	}

	return resp, nil
}

// BatchCancelOrdersByClientID cancels a list of orders by client_order_ids (max 10 per request).
func (c *Client) BatchCancelOrdersByClientID(clientOrderIDs []string) (CancelResponse, error) {
	u, err := url.Parse(c.baseURL + "/v1/client/batch-order")
	if err != nil {
		return CancelResponse{}, fmt.Errorf("failed to parse batch cancel client order url: %w", err)
	}

	q := u.Query()
	q.Set("client_order_ids", strings.Join(clientOrderIDs, ","))
	u.RawQuery = q.Encode()

	req := &rt.Request{
		Method: "DELETE",
		URL:    u.String(),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	c.addAuthHeaders(req, "")

	var resp CancelResponse
	_, sendErr := c.requester.Send(req, &resp)
	if sendErr != nil {
		return CancelResponse{}, fmt.Errorf("failed to cancel /v1/client/batch-order code=%d message=%s: %w", resp.Code, resp.Message, sendErr)
	}
	if !resp.Success {
		return resp, fmt.Errorf("orderly /v1/client/batch-order returned success=false code=%d message=%s", resp.Code, resp.Message)
	}

	return resp, nil
}
