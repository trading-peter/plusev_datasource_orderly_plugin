package main

import (
	"fmt"
	"strings"
	"time"

	ex "github.com/plusev-terminal/go-plugin-common/datasrc/exchange"
	acct "github.com/plusev-terminal/go-plugin-common/datasrc/exchange/account"
	"github.com/plusev-terminal/go-plugin-common/plugin"
)

// handleGetBalances returns balances only (no account metadata beyond balances snapshot).
func (p *OrderlyPlugin) handleGetBalances(params map[string]any) plugin.Response {
	if p.client == nil {
		return plugin.ErrorResponseMsg("client not initialized")
	}

	parsed := ex.GetBalancesParamsFromMap(params)

	wantsCollateral := true
	wantsFutures := true
	if len(parsed.Scopes) > 0 {
		wantsCollateral = false
		wantsFutures = false

		for _, s := range parsed.Scopes {
			switch s {
			case acct.ScopeCollateral:
				wantsCollateral = true
			case acct.ScopeFutures:
				wantsFutures = true
			}
		}
	}

	fetchedAt := time.Now()

	resp := acct.BalancesResponse{
		FetchedAt: fetchedAt,
		Scopes:    map[acct.ScopeType]acct.BalanceScope{},
		Extra:     map[string]any{},
	}

	if wantsCollateral {
		holding, err := p.client.GetClientHolding()
		if err != nil {
			resp.Extra["holdingError"] = err.Error()
		} else {
			if holding.Timestamp > 0 {
				resp.FetchedAt = time.UnixMilli(holding.Timestamp)
			}

			scope := acct.BalanceScope{Balances: map[string]acct.AssetBalance{}, Extra: map[string]any{}}

			for _, h := range holding.Data.Holding {
				asset := strings.ToUpper(strings.TrimSpace(h.Token))
				if asset == "" {
					continue
				}
				scope.Balances[asset] = acct.AssetBalance{
					Asset:            asset,
					Total:            fmt.Sprintf("%.15g", h.Holding),
					AvailableToTrade: fmt.Sprintf("%.15g", h.Holding-h.Frozen),
					Components: map[string]string{
						"frozen": fmt.Sprintf("%.15g", h.Frozen),
					},
					Extra: map[string]any{
						"updatedTime":  h.UpdatedTime,
						"pendingShort": fmt.Sprintf("%.15g", h.PendingShort),
					},
				}
			}
			resp.Scopes[acct.ScopeCollateral] = scope
		}
	}

	if wantsFutures {
		pos, err := p.client.GetAllPositions()
		if err != nil {
			resp.Extra["positionsError"] = err.Error()
		} else {
			if pos.Timestamp > 0 && resp.FetchedAt.IsZero() {
				resp.FetchedAt = time.UnixMilli(pos.Timestamp)
			}

			resp.Scopes[acct.ScopeFutures] = acct.BalanceScope{
				Balances: map[string]acct.AssetBalance{},
				State: &acct.ScopeState{
					Equity:          fmt.Sprintf("%.15g", pos.Data.TotalCollateralValue),
					AvailableMargin: fmt.Sprintf("%.15g", pos.Data.FreeCollateral),
					MarginRatio:     fmt.Sprintf("%.15g", pos.Data.MarginRatio),
					Extra: map[string]any{
						"current_margin_ratio_with_orders":     fmt.Sprintf("%.15g", pos.Data.CurrentMarginRatioWithOrders),
						"open_margin_ratio":                    fmt.Sprintf("%.15g", pos.Data.OpenMarginRatio),
						"initial_margin_ratio":                 fmt.Sprintf("%.15g", pos.Data.InitialMarginRatio),
						"initial_margin_ratio_with_orders":     fmt.Sprintf("%.15g", pos.Data.InitialMarginRatioWithOrders),
						"maintenance_margin_ratio":             fmt.Sprintf("%.15g", pos.Data.MaintenanceMarginRatio),
						"maintenance_margin_ratio_with_orders": fmt.Sprintf("%.15g", pos.Data.MaintenanceMarginRatioWithOrders),
					},
				},
				Extra: map[string]any{},
			}
		}
	}

	return plugin.SuccessResponse(resp, time.Second*3)
}
