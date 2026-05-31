package afi

import (
	"context"
	"testing"
)

func TestDecodeEnvelope(t *testing.T) {
	t.Run("success unmarshals data", func(t *testing.T) {
		var routes []Route
		err := decodeEnvelope([]byte(`{"status":"success","data":[{"path":["0x1","0x2"]}]}`), &routes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(routes) != 1 || len(routes[0].Path) != 2 {
			t.Fatalf("got %+v", routes)
		}
	})

	t.Run("error status surfaces message", func(t *testing.T) {
		err := decodeEnvelope([]byte(`{"status":"error","data":"no route"}`), &[]Route{})
		if err == nil || !contains(err.Error(), "no route") {
			t.Fatalf("want error with message, got %v", err)
		}
	})

	t.Run("empty data is a no-op", func(t *testing.T) {
		var routes []Route
		if err := decodeEnvelope([]byte(`{"status":"success"}`), &routes); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if routes != nil {
			t.Errorf("expected nil routes, got %+v", routes)
		}
	})

	t.Run("malformed envelope errors", func(t *testing.T) {
		if err := decodeEnvelope([]byte(`not-json`), &[]Route{}); err == nil {
			t.Error("expected error")
		}
	})
}

func TestGetRoutes(t *testing.T) {
	srv := newJSONServer(t, 200, map[string]any{
		"status": "success",
		"data":   []map[string]any{{"path": []string{"0xA", "0xB", "0xA"}}},
	})
	defer srv.Close()
	c := &Client{apiURL: srv.URL}

	routes, err := c.GetRoutes(context.Background(), RoutesRequest{"network": "base"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(routes) != 1 || len(routes[0].Path) != 3 {
		t.Fatalf("got %+v", routes)
	}
}

func TestPriceQuote_ReturnsRouteQuotes(t *testing.T) {
	srv := newJSONServer(t, 200, map[string]any{
		"status": "success",
		"data": []map[string]any{
			{"tokenIn": "0xA", "tokenOut": "0xB", "amountInRaw": "100", "amountOutRaw": "105", "routeId": 3, "kind": "uni"},
		},
	})
	defer srv.Close()
	c := &Client{apiURL: srv.URL}

	quotes, err := c.PriceQuote(context.Background(), PriceQuoteRequest{"network": "base"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(quotes) != 1 || quotes[0].Profit().String() != "5" {
		t.Fatalf("got %+v", quotes)
	}
}

func TestFindPath_ReturnsPathQuote(t *testing.T) {
	srv := newJSONServer(t, 200, map[string]any{
		"status": "success",
		"data": map[string]any{
			"network": "base", "path": []string{"0xA", "0xB"},
			"tokenIn": "0xA", "tokenOut": "0xB",
			"amountInRaw": "100", "amountOutRaw": "110", "minOutRaw": "109",
			"steps": "0xabcd",
			"hops":  []map[string]any{{"routeId": 3, "stepData": "0xab"}},
		},
	})
	defer srv.Close()
	c := &Client{apiURL: srv.URL}

	p, err := c.FindPath(context.Background(), PathRequest{"network": "base"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Steps != "0xabcd" || len(p.Hops) != 1 || p.Hops[0].RouteID != 3 {
		t.Fatalf("got %+v", p)
	}
}

func TestGetLiquidationCandidates(t *testing.T) {
	srv := newJSONServer(t, 200, map[string]any{
		"status": "success",
		"data": []map[string]any{
			{"user": "0xU", "debtToken": "USDC", "debtAmount": "500",
				"collaterals": []map[string]any{{"token": "WETH", "balance": "1.5"}}},
		},
	})
	defer srv.Close()
	c := &Client{apiURL: srv.URL}

	pos, err := c.GetLiquidationCandidates(context.Background(), LiquidationCandidatesRequest{"network": "base"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pos) != 1 || pos[0].User != "0xU" || len(pos[0].Collaterals) != 1 {
		t.Fatalf("got %+v", pos)
	}
}

func TestFindArbitrage_TypedRoutes(t *testing.T) {
	srv := newJSONServer(t, 200, map[string]any{
		"status": "success",
		"data": []map[string]any{
			{"tokenIn": "0xA", "tokenOut": "0xA", "amountInRaw": "100", "amountOutRaw": "112", "routeId": 5, "kind": "uni"},
		},
	})
	defer srv.Close()
	c := &Client{apiURL: srv.URL}

	routes, err := c.FindArbitrage(context.Background(), ArbitrageRequest{"tokenIn": "0xA", "tokenOut": "0xA"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(routes) != 1 || routes[0].Profit().String() != "12" {
		t.Fatalf("got %+v", routes)
	}
}

func TestFindArbitrage_ErrorEnvelope(t *testing.T) {
	srv := newJSONServer(t, 200, map[string]any{"status": "error", "data": "no pair"})
	defer srv.Close()
	c := &Client{apiURL: srv.URL}
	if _, err := c.FindArbitrage(context.Background(), ArbitrageRequest{}); err == nil {
		t.Error("expected error from error envelope")
	}
}

func TestQuoteDex_TypedRoutes(t *testing.T) {
	srv := newJSONServer(t, 200, map[string]any{
		"status": "success",
		"data":   []map[string]any{{"tokenIn": "0xA", "tokenOut": "0xB", "amountInRaw": "1", "amountOutRaw": "2", "routeId": 3}},
	})
	defer srv.Close()
	c := &Client{apiURL: srv.URL}

	routes, err := c.QuoteDex(context.Background(), "uniV3", DexQuoteRequest{"network": "base"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(routes) != 1 || routes[0].RouteID != 3 {
		t.Fatalf("got %+v", routes)
	}
}

func TestLiquidate_TypedResult(t *testing.T) {
	srv := newJSONServer(t, 200, map[string]any{
		"status": "success",
		"data": map[string]any{
			"tokenIn": "0xA", "tokenOut": "0xB", "amountIn": "100", "amountOut": "150",
			"profit": "50.0", "steps": "0xbeef",
			"hops": []map[string]any{{"routeId": 10, "kind": "aave"}, {"routeId": 3, "kind": "uni"}},
		},
	})
	defer srv.Close()
	c := &Client{apiURL: srv.URL}

	res, err := c.Liquidate(context.Background(), LiquidateRequest{"pool": "0xP", "user": "0xU"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Profit != "50.0" || res.Steps != "0xbeef" || len(res.Hops) != 2 {
		t.Fatalf("got %+v", res)
	}
}
