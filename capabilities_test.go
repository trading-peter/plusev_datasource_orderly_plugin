package main

import (
	"reflect"
	"testing"

	ex "github.com/plusev-terminal/go-plugin-common/v2/datasrc/exchange"
	acct "github.com/plusev-terminal/go-plugin-common/v2/datasrc/exchange/account"
	tt "github.com/plusev-terminal/go-plugin-common/v2/trading"
)

// TestOrderCapabilities pins the venue declaration the host gates every
// createOrders batch on. It must mirror the create switch in createOrders.go
// exactly: an undeclared feature is rejected before dispatch, while a
// declared-but-ignored field would silently place a partial order.
func TestOrderCapabilities(t *testing.T) {
	caps := (&OrderlyPlugin{}).GetOrderCapabilities()

	if want := []tt.AssetType{tt.AssetTypePerpetual}; !reflect.DeepEqual(caps.AssetTypes, want) {
		t.Errorf("assetTypes = %v, want %v (orderly/markets.go publishes perpetuals only)", caps.AssetTypes, want)
	}

	if want := []acct.OrderType{
		acct.OrderTypeLimit, acct.OrderTypeMarket,
		acct.OrderTypeStop, acct.OrderTypeStopLimit, acct.OrderTypeStopMarket,
		acct.OrderTypeTakeProfit, acct.OrderTypeTakeProfitLimit, acct.OrderTypeTakeProfitMarket,
		acct.OrderTypeTrailingStop,
	}; !reflect.DeepEqual(caps.OrderTypes, want) {
		t.Errorf("orderTypes = %v, want %v (the types the create switch accepts)", caps.OrderTypes, want)
	}

	// GTD stays out: neither order nor algo create body can carry an expiry.
	if want := []acct.TimeInForce{
		acct.TimeInForceGTC, acct.TimeInForceIOC, acct.TimeInForceFOK,
		acct.TimeInForcePostOnly, acct.TimeInForceGTX,
	}; !reflect.DeepEqual(caps.TimeInForce, want) {
		t.Errorf("timeInForce = %v, want %v", caps.TimeInForce, want)
	}

	if want := []acct.TriggerSource{acct.TriggerSourceMark, acct.TriggerSourceLast}; !reflect.DeepEqual(caps.TriggerSources, want) {
		t.Errorf("triggerSources = %v, want %v (orderly serves MARK_PRICE and LAST_PRICE only)", caps.TriggerSources, want)
	}
	if caps.DefaultTriggerSource != acct.TriggerSourceMark {
		t.Errorf("defaultTriggerSource = %q, want %q", caps.DefaultTriggerSource, acct.TriggerSourceMark)
	}

	if !caps.TrailingPercent || caps.TrailingAmount {
		t.Errorf("trailing stops are percent-only: got trailingAmount=%v trailingPercent=%v", caps.TrailingAmount, caps.TrailingPercent)
	}

	for name, declared := range map[string]bool{
		"activationPrice": caps.ActivationPrice,
		"reduceOnly":      caps.ReduceOnly,
		"postOnly":        caps.PostOnly,
		"closePosition":   caps.ClosePosition,
		"quoteQuantity":   caps.QuoteQuantity,
		"clientOrderId":   caps.ClientOrderID,
	} {
		if !declared {
			t.Errorf("%s is implemented by the create path and must be declared", name)
		}
	}

	if caps.PositionSide {
		t.Error("positionSide is never read by the create path and must stay undeclared")
	}
	if caps.CancelGroupOnFailure {
		t.Error("cancelGroupOnFailure is not implemented by the create path and must stay undeclared")
	}

	// ExtraKeys must list exactly the keys the create switch reads — in both
	// spellings the plugin honors.
	wantKeys := map[string]bool{
		"visibleQuantity": true, "visible_quantity": true,
		"slippage":       true,
		"level":          true,
		"postOnlyAdjust": true, "post_only_adjust": true,
		"callbackRate":  true,
		"callbackValue": true,
		"orderTag":      true,
	}
	for _, key := range caps.ExtraKeys {
		if !wantKeys[key] {
			t.Errorf("extra key %q is declared but not read by the create path", key)
		}
		wantKeys[key] = false
	}
	for key, stillMissing := range wantKeys {
		if stillMissing {
			t.Errorf("extra key %q is read by the create path but not declared", key)
		}
	}
}

// TestPickTriggerPriceType pins the trigger-source mapping: Orderly's
// trigger_price_type comes from the first-class TriggerSource field; the
// legacy Extra spelling is intentionally no longer read.
func TestPickTriggerPriceType(t *testing.T) {
	for source, want := range map[acct.TriggerSource]string{
		"":                     "MARK_PRICE",
		acct.TriggerSourceMark: "MARK_PRICE",
		acct.TriggerSourceLast: "LAST_PRICE",
	} {
		order := ex.CreateOrderParams{}
		order.TriggerSource = source
		if got := pickTriggerPriceType(order); got != want {
			t.Errorf("pickTriggerPriceType(triggerSource=%q) = %q, want %q", source, got, want)
		}
	}

	order := ex.CreateOrderParams{Extra: map[string]any{"trigger_price_type": "LAST_PRICE"}}
	if got := pickTriggerPriceType(order); got != "MARK_PRICE" {
		t.Errorf("Extra trigger_price_type must stay unread now that TriggerSource is first-class, got %q", got)
	}
}

// take_profit_limit is declared tradable, so the STOP algo must honor its
// limit price exactly like stop_limit's — otherwise the declaration above
// would let a limit price be silently dropped.
func TestRequiresStopLimit(t *testing.T) {
	for _, typ := range []acct.OrderType{acct.OrderTypeStopLimit, acct.OrderTypeTakeProfitLimit} {
		if !requiresStopLimit(ex.CreateOrderParams{Type: typ}) {
			t.Errorf("%s must place as a limit-priced STOP algo", typ)
		}
	}
	if requiresStopLimit(ex.CreateOrderParams{Type: acct.OrderTypeStopMarket}) {
		t.Error("stop_market must place as a market trigger")
	}
}
