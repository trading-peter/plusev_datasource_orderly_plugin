package main

import (
	"fmt"
	"time"

	ex "github.com/plusev-terminal/go-plugin-common/datasrc/exchange"
	acct "github.com/plusev-terminal/go-plugin-common/datasrc/exchange/account"
	"github.com/plusev-terminal/go-plugin-common/plugin"
)

func (p *OrderlyPlugin) handleGetPositions(params map[string]any) plugin.Response {
	if p.client == nil {
		return plugin.ErrorResponseMsg("client not initialized")
	}

	parsed := ex.GetPositionsParamsFromMap(params)
	if len(parsed.Scopes) > 0 {
		wantsFutures := false
		for _, s := range parsed.Scopes {
			if s == acct.ScopeFutures {
				wantsFutures = true
				break
			}
		}
		if !wantsFutures {
			return plugin.SuccessResponse(acct.PositionsResponse{
				FetchedAt: time.Now(),
				Scopes:    map[acct.ScopeType]acct.PositionScope{},
				Extra:     map[string]any{"note": "scopes filter excluded futures"},
			})
		}
	}

	posResp, err := p.client.GetAllPositions()
	if err != nil {
		return plugin.ErrorResponse(err)
	}

	fetchedAt := time.Now()
	if posResp.Timestamp > 0 {
		fetchedAt = time.UnixMilli(posResp.Timestamp)
	}

	positions := make([]acct.Position, 0, len(posResp.Data.Rows))
	for _, row := range posResp.Data.Rows {
		side := ""
		if row.PositionQty > 0 {
			side = "long"
		} else if row.PositionQty < 0 {
			side = "short"
		}

		estPnL := ""
		if row.PositionQty != 0 && row.MarkPrice != 0 && row.AverageOpenPrice != 0 {
			estPnL = fmt.Sprintf("%.15g", (row.MarkPrice-row.AverageOpenPrice)*row.PositionQty)
		}

		positions = append(positions, acct.Position{
			Symbol:           row.Symbol,
			Side:             side,
			Quantity:         fmt.Sprintf("%.15g", row.PositionQty),
			EntryPrice:       fmt.Sprintf("%.15g", row.AverageOpenPrice),
			MarkPrice:        fmt.Sprintf("%.15g", row.MarkPrice),
			UnrealizedPnL:    estPnL,
			UnsettledPnL:     fmt.Sprintf("%.15g", row.UnsettledPnl),
			LiquidationPrice: fmt.Sprintf("%.15g", row.EstLiqPrice),
			Leverage:         int(row.Leverage),
			IsIsolated:       false,
			Components: map[string]string{
				"pnl_24h":                  fmt.Sprintf("%.15g", row.Pnl24H),
				"fee_24h":                  fmt.Sprintf("%.15g", row.Fee24H),
				"cost_position":            fmt.Sprintf("%.15g", row.CostPosition),
				"settle_price":             fmt.Sprintf("%.15g", row.SettlePrice),
				"pending_long_qty":         fmt.Sprintf("%.15g", row.PendingLongQty),
				"pending_short_qty":        fmt.Sprintf("%.15g", row.PendingShortQty),
				"imr":                      fmt.Sprintf("%.15g", row.Imr),
				"mmr":                      fmt.Sprintf("%.15g", row.Mmr),
				"imr_withdraw_orders":      fmt.Sprintf("%.15g", row.IMRWithdrawOrders),
				"mmr_with_orders":          fmt.Sprintf("%.15g", row.MMRWithOrders),
				"last_sum_unitary_funding": fmt.Sprintf("%.15g", row.LastSumUnitaryFunding),
			},
			Extra: map[string]any{
				"timestamp":    row.Timestamp,
				"updated_time": row.UpdatedTime,
				"seq":          row.Seq,
			},
		})
	}

	scope := acct.PositionScope{
		Positions: positions,
		Extra:     map[string]any{},
	}

	resp := acct.PositionsResponse{
		FetchedAt: fetchedAt,
		Scopes: map[acct.ScopeType]acct.PositionScope{
			acct.ScopeFutures: scope,
		},
		Extra: map[string]any{},
	}

	return plugin.SuccessResponse(resp, time.Second*3)
}
