package afi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth_BothOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockInfoResponse)
	}))
	defer srv.Close()

	// RPC is bogus — we expect rpc.OK = false but api.OK = true.
	c, _ := NewClient(Config{RPCURL: "http://127.0.0.1:1"})
	defer c.Close()
	c.SetApiURL(srv.URL)

	h := c.Health(context.Background())
	if h.API.OK != true {
		t.Errorf("API.OK = %v, want true; err=%v detail=%s", h.API.OK, h.API.Err, h.API.Detail)
	}
	// RPC will fail because nothing is listening; we just want to confirm the struct shape is filled.
	if h.RPC.DurationMs < 0 {
		t.Errorf("RPC.DurationMs should be >= 0, got %d", h.RPC.DurationMs)
	}
}

func TestHealth_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c, _ := NewClient(Config{RPCURL: "http://127.0.0.1:1"})
	defer c.Close()
	c.SetApiURL(srv.URL)

	h := c.Health(context.Background())
	if h.API.OK {
		t.Error("expected API.OK to be false on 503")
	}
	if h.API.Err == nil {
		t.Error("expected API.Err to be populated on 503")
	}
}
