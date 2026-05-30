package afi

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestMetadataCache_BasicGetSet(t *testing.T) {
	c := newMetadataCache()
	token := common.HexToAddress("0xaaaa589fcd6edb6e08f4c7c32d4f71b54bda02913")

	if _, ok := c.get(token); ok {
		t.Error("expected miss on empty cache")
	}

	c.set(token, TokenMetadata{Symbol: "USDC", Name: "USD Coin", Decimals: 6})
	got, ok := c.get(token)
	if !ok {
		t.Fatal("expected hit after set")
	}
	if got.Symbol != "USDC" || got.Decimals != 6 {
		t.Errorf("bad metadata: %+v", got)
	}
}

func TestMetadataCache_Clear(t *testing.T) {
	c := newMetadataCache()
	token := common.HexToAddress("0xaaaa")
	c.set(token, TokenMetadata{Symbol: "X"})
	c.clear()
	if _, ok := c.get(token); ok {
		t.Error("expected miss after clear")
	}
}

func TestMetadataCache_NilSafe(t *testing.T) {
	// Operations on a nil cache should be no-ops, not panics.
	var c *metadataCache
	c.set(common.HexToAddress("0xaaaa"), TokenMetadata{Symbol: "X"})
	if _, ok := c.get(common.HexToAddress("0xaaaa")); ok {
		t.Error("nil cache should always miss")
	}
	c.clear()
}

func TestClient_ClearTokenMetadataCache_DoesNotPanic(t *testing.T) {
	c, _ := NewClient(Config{RPCURL: "http://localhost:1"})
	defer c.Close()
	c.ClearTokenMetadataCache()
}
