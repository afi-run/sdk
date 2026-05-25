package afi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

var mockInfoResponse = map[string]any{
	"status": "success",
	"data": map[string]any{
		"base": []map[string]any{
			{"address": "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913", "symbol": "USDC", "decimals": 6, "active": true},
			{"address": "0x4200000000000000000000000000000000000006", "symbol": "WETH", "decimals": 18, "active": true},
			{"address": "0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf", "symbol": "cbBTC", "decimals": 8, "active": true},
		},
	},
	"time": "0ms",
}

func TestFetchTokens(t *testing.T) {
	t.Run("returns base tokens on success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockInfoResponse)
		}))
		defer srv.Close()

		tokens, err := fetchTokensFrom(context.Background(), srv.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(tokens) != 3 {
			t.Fatalf("expected 3 tokens, got %d", len(tokens))
		}

		usdc := tokens[0]
		if usdc.Symbol != "USDC" {
			t.Errorf("tokens[0].Symbol = %q, want USDC", usdc.Symbol)
		}
		if usdc.Decimals != 6 {
			t.Errorf("tokens[0].Decimals = %d, want 6", usdc.Decimals)
		}
		if !usdc.Active {
			t.Error("tokens[0].Active should be true")
		}

		weth := tokens[1]
		if weth.Symbol != "WETH" || weth.Decimals != 18 {
			t.Errorf("tokens[1] = %+v, want WETH/18", weth)
		}
	})

	t.Run("returns error on HTTP 500", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		_, err := fetchTokensFrom(context.Background(), srv.URL)
		if err == nil {
			t.Fatal("expected error for HTTP 500, got nil")
		}
	})

	t.Run("returns error when status is not success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"status": "error"})
		}))
		defer srv.Close()

		_, err := fetchTokensFrom(context.Background(), srv.URL)
		if err == nil {
			t.Fatal("expected error for non-success status, got nil")
		}
	})

	t.Run("returns error when base key is missing", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"status": "success",
				"data":   map[string]any{"arbitrum": []any{}},
			})
		}))
		defer srv.Close()

		_, err := fetchTokensFrom(context.Background(), srv.URL)
		if err == nil {
			t.Fatal("expected error for missing base key, got nil")
		}
	})

	t.Run("context cancellation returns error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(mockInfoResponse)
		}))
		defer srv.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately

		_, err := fetchTokensFrom(ctx, srv.URL)
		if err == nil {
			t.Fatal("expected error for cancelled context, got nil")
		}
	})

	t.Run("returns error for invalid URL", func(t *testing.T) {
		_, err := fetchTokensFrom(context.Background(), "://invalid-url")
		if err == nil {
			t.Fatal("expected error for invalid URL, got nil")
		}
	})

	t.Run("returns error when response body is not valid JSON", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("not-json{{{"))
		}))
		defer srv.Close()

		_, err := fetchTokensFrom(context.Background(), srv.URL)
		if err == nil {
			t.Fatal("expected error for invalid JSON, got nil")
		}
	})
}
