package main

import (
	"fmt"
	"strings"
	"time"

	ex "github.com/plusev-terminal/go-plugin-common/datasrc/exchange"
	acct "github.com/plusev-terminal/go-plugin-common/datasrc/exchange/account"
	"github.com/plusev-terminal/go-plugin-common/plugin"
)

func (p *OrderlyPlugin) handleCancelOrders(params map[string]any) plugin.Response {
	if p.client == nil {
		return plugin.ErrorResponseMsg("client not initialized")
	}

	parsed := ex.CancelOrdersParamsFromMap(params)
	if err := parsed.Validate(); err != nil {
		return plugin.ErrorResponse(err)
	}

	symbol := strings.TrimSpace(parsed.Market.Symbol)

	results := make([]acct.CancelOrderResult, 0)

	if len(parsed.IDs) == 0 {
		res := acct.CancelOrderResult{Index: 0}

		if _, err := p.client.CancelAllPendingOrders(symbol); err != nil {
			res.Success = false
			res.Error = err.Error()
		} else {
			res.Success = true
		}

		res.Extra = map[string]any{"scope": "orders"}
		results = append(results, res)

		algoRes := acct.CancelOrderResult{Index: 1}

		if _, err := p.client.CancelAllPendingAlgoOrders(symbol); err != nil {
			algoRes.Success = false
			algoRes.Error = err.Error()
		} else {
			algoRes.Success = true
		}

		algoRes.Extra = map[string]any{"scope": "algo_orders"}
		results = append(results, algoRes)

		resp := acct.CancelOrdersResponse{
			SubmittedAt: time.Now(),
			Results:     results,
		}

		return plugin.SuccessResponse(resp)
	}

	for i, rawID := range parsed.IDs {
		id := strings.TrimSpace(rawID)
		res := acct.CancelOrderResult{Index: i}

		if id == "" {
			res.Success = false
			res.Error = "id is required"
			results = append(results, res)
			continue
		}

		isNumeric := true

		for _, r := range id {
			if r < '0' || r > '9' {
				isNumeric = false
				break
			}
		}

		if isNumeric {
			res.OrderID = id

			_, err := p.client.CancelOrder(id, symbol)
			if err != nil {
				_, algoErr := p.client.CancelAlgoOrder(id, symbol)
				if algoErr == nil {
					res.Success = true
					results = append(results, res)
					continue
				}

				res.Success = false
				res.Error = fmt.Sprintf("cancel order failed: %v; cancel algo failed: %v", err, algoErr)
				results = append(results, res)
				continue
			}

			res.Success = true
			results = append(results, res)
			continue
		}

		res.ClientOrderID = id

		_, err := p.client.CancelOrderByClientID(id, symbol)
		if err != nil {
			_, algoErr := p.client.CancelAlgoOrderByClientID(id, symbol)
			if algoErr == nil {
				res.Success = true
				results = append(results, res)
				continue
			}

			res.Success = false
			res.Error = fmt.Sprintf("cancel order failed: %v; cancel algo failed: %v", err, algoErr)
			results = append(results, res)
			continue
		}

		res.Success = true
		results = append(results, res)
	}

	resp := acct.CancelOrdersResponse{
		SubmittedAt: time.Now(),
		Results:     results,
	}

	return plugin.SuccessResponse(resp)
}
