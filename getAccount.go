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

// handleGetAccount returns normalized account details and balances.
func (p *OrderlyPlugin) handleGetAccount(params map[string]any) plugin.Response {
	if p.client == nil {
		return plugin.ErrorResponseMsg("client not initialized")
	}

	parsed := ex.GetAccountParamsFromMap(params)

	wantsSpot := true
	wantsFutures := true
	if len(parsed.Scopes) > 0 {
		wantsSpot = false
		wantsFutures = false

		for _, s := range parsed.Scopes {
			switch s {
			case acct.ScopeSpot:
				wantsSpot = true
			case acct.ScopeFutures:
				wantsFutures = true
			}
		}
	}

	info, err := p.client.GetClientInfo()
	if err != nil {
		return plugin.ErrorResponse(err)
	}

	var holdingErr error
	var holding orderly.ClientHoldingResponse

	if wantsSpot {
		holding, holdingErr = p.client.GetClientHolding()
	}

	var posErr error
	var posResp orderly.PositionsResponse

	if wantsFutures {
		posResp, posErr = p.client.GetAllPositions()
	}

	fetchedAt := time.Now()
	if info.Timestamp > 0 {
		fetchedAt = time.UnixMilli(info.Timestamp)
	}

	collateralScope := acct.BalanceScope{
		Balances: map[string]acct.AssetBalance{},
		Extra:    map[string]any{},
	}

	if holdingErr == nil {
		for _, h := range holding.Data.Holding {
			if strings.TrimSpace(h.Token) == "" {
				continue
			}
			asset := strings.ToUpper(strings.TrimSpace(h.Token))
			collateralScope.Balances[asset] = acct.AssetBalance{
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
	} else if wantsSpot {
		collateralScope.Extra["holdingError"] = holdingErr.Error()
	}

	scopes := map[acct.ScopeType]acct.BalanceScope{}

	if wantsSpot {
		scopes[acct.ScopeCollateral] = collateralScope
	}

	if wantsFutures {
		fScope := acct.BalanceScope{
			Balances: map[string]acct.AssetBalance{},
			State:    &acct.ScopeState{},
			Extra:    map[string]any{},
		}

		if posErr == nil {
			fScope.State = &acct.ScopeState{
				Equity:          fmt.Sprintf("%.15g", posResp.Data.TotalCollateralValue),
				AvailableMargin: fmt.Sprintf("%.15g", posResp.Data.FreeCollateral),
				MarginRatio:     fmt.Sprintf("%.15g", posResp.Data.MarginRatio),
				Extra: map[string]any{
					"current_margin_ratio_with_orders":     fmt.Sprintf("%.15g", posResp.Data.CurrentMarginRatioWithOrders),
					"open_margin_ratio":                    fmt.Sprintf("%.15g", posResp.Data.OpenMarginRatio),
					"initial_margin_ratio":                 fmt.Sprintf("%.15g", posResp.Data.InitialMarginRatio),
					"initial_margin_ratio_with_orders":     fmt.Sprintf("%.15g", posResp.Data.InitialMarginRatioWithOrders),
					"maintenance_margin_ratio":             fmt.Sprintf("%.15g", posResp.Data.MaintenanceMarginRatio),
					"maintenance_margin_ratio_with_orders": fmt.Sprintf("%.15g", posResp.Data.MaintenanceMarginRatioWithOrders),
				},
			}
		} else {
			fScope.Extra["positionsError"] = posErr.Error()
		}
		scopes[acct.ScopeFutures] = fScope
	}

	acctResp := acct.Account{
		Exchange:     "orderly",
		AccountID:    info.Data.AccountID,
		Type:         acct.AccountTypeFutures,
		FetchedAt:    fetchedAt,
		MaxLeverage:  int(info.Data.MaxLeverage),
		CustodyModel: acct.CustodyModelExchangeCustody,
		IsCustodial:  true,
		Scopes:       scopes,
		Raw:          nil,
		Extra: map[string]any{
			"email":                   info.Data.Email,
			"accountMode":             info.Data.AccountMode,
			"maintenanceCancelOrders": info.Data.MaintenanceCancelOrders,
			"imrFactor":               info.Data.ImrFactor,
			"maxNotional":             info.Data.MaxNotional,
		},
	}

	return plugin.SuccessResponse(acctResp, time.Second*5)
}
