package afi

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
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
				"tokenIn":       usdc.Hex(),
				"tokenOut":      weth.Hex(),
				"amountIn":      "1000",
				"amountOut":     "0.5",
				"minOut":        "0.4975",
				"amountInRaw":   "1000000000",
				"amountOutRaw":  "500000000000000000",
				"minOutRaw":     "497500000000000000",
				"steps":         "0xabcdef",
				"slippage":      0.5,
				"path":          []string{usdc.Hex(), weth.Hex()},
				"tokenInPrice":  "0.00047147437880193935",
				"tokenOutPrice": "2121.006029089203",
			},
		}
	}

	baseOpts := &quoteOptions{
		tokenIn:  usdc.Hex(),
		tokenOut: weth.Hex(),
		amountIn: "1000",
		slippage: 0.5,
		maxHops:  2,
		network:  NetworkBase,
	}

	t.Run("returns valid quote on success", func(t *testing.T) {
		srv := newJSONServer(t, 200, makeSuccessPayload())
		defer srv.Close()

		q, err := fetchQuoteFrom(context.Background(), baseOpts, 35, srv.URL)
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
		if q.TokenInPrice != "0.00047147437880193935" {
			t.Errorf("TokenInPrice = %q", q.TokenInPrice)
		}
		if q.TokenOutPrice != "2121.006029089203" {
			t.Errorf("TokenOutPrice = %q", q.TokenOutPrice)
		}
	})

	t.Run("sends correct request body", func(t *testing.T) {
		var captured map[string]any
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&captured)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(makeSuccessPayload())
		}))
		defer srv.Close()

		opts := &quoteOptions{
			tokenIn:  usdc.Hex(),
			tokenOut: weth.Hex(),
			amountIn: "1000",
			slippage: 0.5,
			maxHops:  2,
			network:  NetworkBase,
			rpcUrls:  []RpcUrlInfo{{URL: "https://rpc.example.com"}},
		}
		fetchQuoteFrom(context.Background(), opts, 35, srv.URL)

		if captured["network"] != "base" {
			t.Errorf("network = %v, want base", captured["network"])
		}
		if captured["amountIn"] != "1000" {
			t.Errorf("amountIn = %v, want 1000", captured["amountIn"])
		}
		if captured["show"] != true {
			t.Errorf("show = %v, want true", captured["show"])
		}
		rpcUrls, ok := captured["rpcUrls"].([]any)
		if !ok || len(rpcUrls) == 0 {
			t.Error("rpcUrls should be a non-empty array")
		}
	})

	t.Run("includes priceBase when set", func(t *testing.T) {
		var captured map[string]any
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&captured)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(makeSuccessPayload())
		}))
		defer srv.Close()

		opts := &quoteOptions{
			tokenIn: usdc.Hex(), tokenOut: weth.Hex(), amountIn: "1000",
			slippage: 0.5, maxHops: 2, network: NetworkBase,
			priceBase: "USDC",
		}
		fetchQuoteFrom(context.Background(), opts, 35, srv.URL)

		if captured["priceBase"] != "USDC" {
			t.Errorf("priceBase = %v, want USDC", captured["priceBase"])
		}
	})

	t.Run("includes dexs when set", func(t *testing.T) {
		var captured map[string]any
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&captured)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(makeSuccessPayload())
		}))
		defer srv.Close()

		opts := &quoteOptions{
			tokenIn: usdc.Hex(), tokenOut: weth.Hex(), amountIn: "1000",
			slippage: 0.5, maxHops: 2, network: NetworkBase,
			dexs: []Dex{DexUniV3, DexAerodrome},
		}
		fetchQuoteFrom(context.Background(), opts, 35, srv.URL)

		dexs, ok := captured["dexs"].([]any)
		if !ok || len(dexs) != 2 {
			t.Errorf("dexs = %v, want 2 entries", captured["dexs"])
		}
	})

	t.Run("maps tokenInBasePrice and tokenOutBasePrice from response", func(t *testing.T) {
		payload := makeSuccessPayload()
		payload["data"].(map[string]any)["tokenInBasePrice"] = "1.00"
		payload["data"].(map[string]any)["tokenOutBasePrice"] = "2121.00"
		srv := newJSONServer(t, 200, payload)
		defer srv.Close()

		q, err := fetchQuoteFrom(context.Background(), baseOpts, 35, srv.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if q.TokenInBasePrice != "1.00" {
			t.Errorf("TokenInBasePrice = %q, want 1.00", q.TokenInBasePrice)
		}
		if q.TokenOutBasePrice != "2121.00" {
			t.Errorf("TokenOutBasePrice = %q, want 2121.00", q.TokenOutBasePrice)
		}
	})

	t.Run("uses BSC network when set", func(t *testing.T) {
		var captured map[string]any
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&captured)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(makeSuccessPayload())
		}))
		defer srv.Close()

		opts := &quoteOptions{
			tokenIn: usdc.Hex(), tokenOut: weth.Hex(), amountIn: "1000",
			slippage: 0.5, maxHops: 2, network: NetworkBSC,
		}
		fetchQuoteFrom(context.Background(), opts, 35, srv.URL)

		if captured["network"] != "bsc" {
			t.Errorf("network = %v, want bsc", captured["network"])
		}
	})

	t.Run("sends amountIn string directly to quoter", func(t *testing.T) {
		var capturedBody map[string]any
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&capturedBody)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(makeSuccessPayload())
		}))
		defer srv.Close()

		opts := &quoteOptions{
			tokenIn: usdc.Hex(), tokenOut: weth.Hex(), amountIn: "1.5",
			slippage: 0.5, maxHops: 2, network: NetworkBase,
		}
		fetchQuoteFrom(context.Background(), opts, 35, srv.URL)

		if capturedBody["amountIn"] != "1.5" {
			t.Errorf("amountIn = %v, want \"1.5\"", capturedBody["amountIn"])
		}
	})

	t.Run("returns error when status is not success", func(t *testing.T) {
		srv := newJSONServer(t, 200, map[string]any{"status": "error", "message": "no route"})
		defer srv.Close()

		_, err := fetchQuoteFrom(context.Background(), baseOpts, 35, srv.URL)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error when minOutRaw is zero", func(t *testing.T) {
		payload := makeSuccessPayload()
		payload["data"].(map[string]any)["minOutRaw"] = "0"
		srv := newJSONServer(t, 200, payload)
		defer srv.Close()

		_, err := fetchQuoteFrom(context.Background(), baseOpts, 35, srv.URL)
		if err == nil {
			t.Fatal("expected error for zero minOut, got nil")
		}
	})

	t.Run("returns error on HTTP 500", func(t *testing.T) {
		srv := newJSONServer(t, 500, nil)
		defer srv.Close()

		_, err := fetchQuoteFrom(context.Background(), baseOpts, 35, srv.URL)
		if err == nil {
			t.Fatal("expected error for HTTP 500, got nil")
		}
	})

	t.Run("uses fallback message when error has no message field", func(t *testing.T) {
		srv := newJSONServer(t, 200, map[string]any{"status": "error"})
		defer srv.Close()

		_, err := fetchQuoteFrom(context.Background(), baseOpts, 35, srv.URL)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() == "" {
			t.Error("error message should not be empty")
		}
	})

	t.Run("extracts message from string data on error", func(t *testing.T) {
		srv := newJSONServer(t, 200, map[string]any{"status": "error", "data": "insufficient liquidity"})
		defer srv.Close()

		_, err := fetchQuoteFrom(context.Background(), baseOpts, 35, srv.URL)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "insufficient liquidity") {
			t.Errorf("expected error to contain %q, got %q", "insufficient liquidity", err.Error())
		}
	})

	t.Run("returns error on context cancellation", func(t *testing.T) {
		srv := newJSONServer(t, 200, makeSuccessPayload())
		defer srv.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := fetchQuoteFrom(ctx, baseOpts, 35, srv.URL)
		if err == nil {
			t.Fatal("expected error for cancelled context, got nil")
		}
	})

	t.Run("returns error for invalid quoter URL", func(t *testing.T) {
		_, err := fetchQuoteFrom(context.Background(), baseOpts, 35, "://invalid-url")
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

		_, err := fetchQuoteFrom(context.Background(), baseOpts, 35, srv.URL)
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
