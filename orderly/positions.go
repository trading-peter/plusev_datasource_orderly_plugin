package orderly

import (
	"fmt"

	rt "github.com/plusev-terminal/go-plugin-common/requester/types"
)

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
		return PositionsResponse{}, fmt.Errorf("failed to fetch /v1/positions code=%d message=%s: %w", resp.Code, resp.Message, err)
	}
	if !resp.Success {
		return PositionsResponse{}, fmt.Errorf("orderly /v1/positions returned success=false code=%d message=%s", resp.Code, resp.Message)
	}
	return resp, nil
}
