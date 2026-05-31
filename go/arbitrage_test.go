package afi

import (
	"math/big"
	"testing"
)

func sampleRoute() RouteQuote {
	return RouteQuote{
		Network:      "base",
		Kind:         "uni",
		TokenIn:      "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
		TokenOut:     "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
		AmountIn:     "1000",
		AmountInRaw:  "1000000000",
		AmountOut:    "1005",
		AmountOutRaw: "1005000000",
		MinOut:       "1000",
		MinOutRaw:    "1000000000",
		RouteID:      3,
		StepData:     "0xdeadbeef",
	}
}

func TestRouteQuote_Profit(t *testing.T) {
	if got := sampleRoute().Profit(); got.String() != "5000000" {
		t.Errorf("profit = %s, want 5000000", got)
	}
	bad := RouteQuote{AmountInRaw: "x", AmountOutRaw: "1"}
	if bad.Profit() != nil {
		t.Error("expected nil profit on unparseable amounts")
	}
}

func TestQuoteFromRoute(t *testing.T) {
	q, err := QuoteFromRoute(sampleRoute(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.TokenIn != q.TokenOut {
		t.Error("expected a cycle (TokenIn == TokenOut)")
	}
	if q.AmountInWei.String() != "1000000000" {
		t.Errorf("amountInWei = %s, want 1000000000", q.AmountInWei)
	}
	if q.MinOutWei.String() != "1000000000" {
		t.Errorf("minOutWei = %s, want route MinOutRaw", q.MinOutWei)
	}
	if q.Network != NetworkBase {
		t.Errorf("network = %s, want base", q.Network)
	}
	// Steps must be EncodeSteps of the single hop: 1 step, id=3, 4 data bytes.
	want, _ := EncodeSteps([]Step{{ID: 3, Data: []byte{0xde, 0xad, 0xbe, 0xef}}})
	if string(q.Steps) != string(want) {
		t.Errorf("steps = %x, want %x", q.Steps, want)
	}
}

func TestQuoteFromRoute_MinOutOverride(t *testing.T) {
	override := big.NewInt(1_002_000_000)
	q, err := QuoteFromRoute(sampleRoute(), override)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.MinOutWei.Cmp(override) != 0 {
		t.Errorf("minOutWei = %s, want override %s", q.MinOutWei, override)
	}
}

func TestQuoteFromRoute_Errors(t *testing.T) {
	bad := sampleRoute()
	bad.AmountInRaw = "not-a-number"
	if _, err := QuoteFromRoute(bad, nil); err == nil {
		t.Error("expected error on bad amountInRaw")
	}

	overflow := sampleRoute()
	overflow.RouteID = 70000 // > uint16 max
	if _, err := QuoteFromRoute(overflow, big.NewInt(1)); err == nil {
		t.Error("expected error on routeId overflow")
	}
}
