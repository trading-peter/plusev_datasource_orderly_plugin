package orderly

import (
	"fmt"

	rt "github.com/plusev-terminal/go-plugin-common/requester/types"
)

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
