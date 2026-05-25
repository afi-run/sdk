package afi

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestFormatAmount(t *testing.T) {
	tests := []struct {
		name     string
		amount   *big.Int
		decimals uint8
		want     string
	}{
		{"whole USDC", big.NewInt(1000_000000), 6, "1000"},
		{"fractional 1.5 USDC", big.NewInt(1_500000), 6, "1.5"},
		{"trims trailing zeros", big.NewInt(1_100000), 6, "1.1"},
		{"less than 1 token", big.NewInt(123456), 6, "0.123456"},
		{"whole WETH 18 dec", new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil), 18, "1"},
		{"half WETH", new(big.Int).Div(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil), big.NewInt(2)), 18, "0.5"},
		{"zero", big.NewInt(0), 6, "0"},
		{"large whole", big.NewInt(1000000_000000), 6, "1000000"},
		{"preserves all decimals", big.NewInt(1_000001), 6, "1.000001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatAmount(tt.amount, tt.decimals)
			if got != tt.want {
				t.Errorf("formatAmount(%s, %d) = %q, want %q", tt.amount, tt.decimals, got, tt.want)
			}
		})
	}
}

func TestFetchQuote(t *testing.T) {
	usdc := common.HexToAddress("0x833589fcd6edb6e08f4c7c32d4f71b54bda02913")
	weth := common.HexToAddress("0x4200000000000000000000000000000000000006")

	makeSuccessPayload := func() map[string]any {
		return map[string]any{
			"status": "success",
			"data": map[string]any{
				"tokenIn":      usdc.Hex(),
				"tokenOut":     weth.Hex(),
				"amountInRaw":  "1000000000",
				"amountOutRaw": "500000000000000000",
				"minOutRaw":    "497500000000000000",
				"steps":        "0xabcdef",
				"slippage":     0.5,
				"path":         []string{usdc.Hex(), weth.Hex()},
			},
		}
	}

	baseParams := SwapParams{
		TokenIn:  usdc,
		TokenOut: weth,
		AmountIn: big.NewInt(1000_000000),
		Slippage: 0.5,
	}

	t.Run("returns valid quote on success", func(t *testing.T) {
		srv := newJSONServer(t, 200, makeSuccessPayload())
		defer srv.Close()

		q, err := fetchQuoteFrom(context.Background(), baseParams, 6, 35, "https://rpc.example.com", srv.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := big.NewInt(1000000000)
		if q.AmountInWei.Cmp(want) != 0 {
			t.Errorf("AmountInWei = %s, want %s", q.AmountInWei, want)
		}
		if q.MinOutWei.Cmp(big.NewInt(0)) <= 0 {
			t.Error("MinOutWei should be > 0")
		}
		if q.FeeBps != 35 {
			t.Errorf("FeeBps = %d, want 35", q.FeeBps)
		}
		if len(q.Steps) == 0 {
			t.Error("Steps should not be empty")
		}
		if len(q.Path) != 2 {
			t.Errorf("Path length = %d, want 2", len(q.Path))
		}
	})

	t.Run("sends formatted amountIn not raw wei", func(t *testing.T) {
		var capturedBody map[string]any
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&capturedBody)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(makeSuccessPayload())
		}))
		defer srv.Close()

		params := SwapParams{TokenIn: usdc, TokenOut: weth, AmountIn: big.NewInt(1_500000), Slippage: 0.5}
		fetchQuoteFrom(context.Background(), params, 6, 35, "https://rpc.example.com", srv.URL)

		if capturedBody["amountIn"] != "1.5" {
			t.Errorf("amountIn = %v, want \"1.5\"", capturedBody["amountIn"])
		}
		if capturedBody["network"] != "base" {
			t.Errorf("network = %v, want \"base\"", capturedBody["network"])
		}
	})

	t.Run("returns error when status is not success", func(t *testing.T) {
		srv := newJSONServer(t, 200, map[string]any{"status": "error", "message": "no route"})
		defer srv.Close()

		_, err := fetchQuoteFrom(context.Background(), baseParams, 6, 35, "https://rpc.example.com", srv.URL)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error when minOutRaw is zero", func(t *testing.T) {
		payload := makeSuccessPayload()
		payload["data"].(map[string]any)["minOutRaw"] = "0"
		srv := newJSONServer(t, 200, payload)
		defer srv.Close()

		_, err := fetchQuoteFrom(context.Background(), baseParams, 6, 35, "https://rpc.example.com", srv.URL)
		if err == nil {
			t.Fatal("expected error for zero minOut, got nil")
		}
	})

	t.Run("returns error on HTTP 500", func(t *testing.T) {
		srv := newJSONServer(t, 500, nil)
		defer srv.Close()

		_, err := fetchQuoteFrom(context.Background(), baseParams, 6, 35, "https://rpc.example.com", srv.URL)
		if err == nil {
			t.Fatal("expected error for HTTP 500, got nil")
		}
	})

	t.Run("uses fallback message when error has no message field", func(t *testing.T) {
		srv := newJSONServer(t, 200, map[string]any{"status": "error"})
		defer srv.Close()

		_, err := fetchQuoteFrom(context.Background(), baseParams, 6, 35, "https://rpc.example.com", srv.URL)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() == "" {
			t.Error("error message should not be empty")
		}
	})

	t.Run("returns error on context cancellation", func(t *testing.T) {
		srv := newJSONServer(t, 200, makeSuccessPayload())
		defer srv.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := fetchQuoteFrom(ctx, baseParams, 6, 35, "https://rpc.example.com", srv.URL)
		if err == nil {
			t.Fatal("expected error for cancelled context, got nil")
		}
	})

	t.Run("returns error for invalid quoter URL", func(t *testing.T) {
		_, err := fetchQuoteFrom(context.Background(), baseParams, 6, 35, "https://rpc.example.com", "://invalid-url")
		if err == nil {
			t.Fatal("expected error for invalid URL, got nil")
		}
	})

	t.Run("returns error when response body is not valid JSON", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("not-json{{{"))
		}))
		defer srv.Close()

		_, err := fetchQuoteFrom(context.Background(), baseParams, 6, 35, "https://rpc.example.com", srv.URL)
		if err == nil {
			t.Fatal("expected error for invalid JSON, got nil")
		}
	})
}

func newJSONServer(t *testing.T, status int, body any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if body != nil {
			json.NewEncoder(w).Encode(body)
		}
	}))
}
