package orderly

import (
	"fmt"

	rt "github.com/plusev-terminal/go-plugin-common/requester/types"
)

type ClientInfoResponse struct {
	Success   bool  `json:"success"`
	Timestamp int64 `json:"timestamp"`
	Data      struct {
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

func (c *Client) GetClientInfo() (ClientInfoResponse, error) {
	req := &rt.Request{
		Method: "GET",
		URL:    c.baseURL + "/v1/client/info",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
	c.addAuthHeaders(req, "")

	var resp ClientInfoResponse
	_, err := c.requester.Send(req, &resp)
	if err != nil {
		return ClientInfoResponse{}, fmt.Errorf("failed to fetch /v1/client/info: %w", err)
	}
	if !resp.Success {
		return ClientInfoResponse{}, fmt.Errorf("orderly /v1/client/info returned success=false")
	}
	return resp, nil
}

func (c *Client) GetClientHolding() (ClientHoldingResponse, error) {
	req := &rt.Request{
		Method: "GET",
		URL:    c.baseURL + "/v1/client/holding",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
	c.addAuthHeaders(req, "")

	var resp ClientHoldingResponse
	_, err := c.requester.Send(req, &resp)

	if err != nil {
		return ClientHoldingResponse{}, fmt.Errorf("failed to fetch /v1/client/holding: %w", err)
	}

	if !resp.Success {
		return ClientHoldingResponse{}, fmt.Errorf("orderly /v1/client/holding returned success=false")
	}

	return resp, nil
}

func (c *Client) GetAllPositions() (PositionsResponse, error) {
	req := &rt.Request{
		Method: "GET",
		URL:    c.baseURL + "/v1/positions",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
	c.addAuthHeaders(req, "")

	var resp PositionsResponse
	_, err := c.requester.Send(req, &resp)
	if err != nil {
		return PositionsResponse{}, fmt.Errorf("failed to fetch /v1/positions: %w", err)
	}
	if !resp.Success {
		return PositionsResponse{}, fmt.Errorf("orderly /v1/positions returned success=false")
	}
	return resp, nil
}
