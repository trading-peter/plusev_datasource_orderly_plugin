package orderly

import (
	"fmt"

	rt "github.com/plusev-terminal/go-plugin-common/requester/types"
)

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
		return ClientInfoResponse{}, fmt.Errorf("failed to fetch /v1/client/info code=%d message=%s: %w", resp.Code, resp.Message, err)
	}
	if !resp.Success {
		return ClientInfoResponse{}, fmt.Errorf("orderly /v1/client/info returned success=false code=%d message=%s", resp.Code, resp.Message)
	}
	return resp, nil
}
