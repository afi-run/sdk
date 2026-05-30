package afi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func mockInfoServer(t *testing.T, calls *atomic.Int32) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls != nil {
			calls.Add(1)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockInfoResponse)
	}))
}

func TestGetTokens_CachesPerNetwork(t *testing.T) {
	var calls atomic.Int32
	srv := mockInfoServer(t, &calls)
	defer srv.Close()

	c, _ := NewClient(Config{RPCURL: "http://localhost:1"})
	defer c.Close()
	c.SetApiURL(srv.URL)

	// /info?network=base will be hit by GetTokens; the mock returns base/* anyway.
	ctx := context.Background()
	tokens1, err := c.GetTokens(ctx)
	if err != nil {
		t.Fatalf("first GetTokens: %v", err)
	}
	tokens2, err := c.GetTokens(ctx)
	if err != nil {
		t.Fatalf("second GetTokens: %v", err)
	}

	if calls.Load() != 1 {
		t.Errorf("expected exactly 1 RPC hit thanks to cache, got %d", calls.Load())
	}
	if len(tokens1) != len(tokens2) {
		t.Errorf("cached slice length mismatch: %d vs %d", len(tokens1), len(tokens2))
	}
}

func TestClearTokensCache_ForcesRefresh(t *testing.T) {
	var calls atomic.Int32
	srv := mockInfoServer(t, &calls)
	defer srv.Close()

	c, _ := NewClient(Config{RPCURL: "http://localhost:1"})
	defer c.Close()
	c.SetApiURL(srv.URL)
	ctx := context.Background()

	if _, err := c.GetTokens(ctx); err != nil {
		t.Fatal(err)
	}
	c.ClearTokensCache()
	if _, err := c.GetTokens(ctx); err != nil {
		t.Fatal(err)
	}

	if calls.Load() != 2 {
		t.Errorf("expected 2 hits after ClearTokensCache, got %d", calls.Load())
	}
}

func TestFindToken_CaseInsensitive(t *testing.T) {
	srv := mockInfoServer(t, nil)
	defer srv.Close()

	c, _ := NewClient(Config{RPCURL: "http://localhost:1"})
	defer c.Close()
	c.SetApiURL(srv.URL)
	ctx := context.Background()

	tok, err := c.FindToken(ctx, "usdc")
	if err != nil {
		t.Fatalf("FindToken: %v", err)
	}
	if tok == nil {
		t.Fatal("expected USDC to be found case-insensitively")
	}
	if !strings.EqualFold(tok.Symbol, "USDC") {
		t.Errorf("got symbol %q", tok.Symbol)
	}
}

func TestFindToken_ReturnsNilWhenMissing(t *testing.T) {
	srv := mockInfoServer(t, nil)
	defer srv.Close()

	c, _ := NewClient(Config{RPCURL: "http://localhost:1"})
	defer c.Close()
	c.SetApiURL(srv.URL)

	tok, err := c.FindToken(context.Background(), "DOGE")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if tok != nil {
		t.Errorf("expected nil for missing token, got %+v", tok)
	}
}
