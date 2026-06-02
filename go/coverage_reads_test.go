package afi

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// readsServer returns a fake RPC that answers every eth_call with the uint256
// word `1` — which decodes as bool=true, uint16/uint8=1, or address 0x…01, so a
// single handler satisfies every read method.
func readsServer(t *testing.T) (*Client, func()) {
	t.Helper()
	srv := newRPCServer(t, rpcHandlers{
		chainID: 8453,
		ethCall: func(common.Address, []byte) []byte { return encodeUint256(big.NewInt(1)) },
	})
	c := mustClientWith(t, srv.URL, testPrivKey)
	return c, func() { c.Close(); srv.Close() }
}

func TestReads_HappyPath(t *testing.T) {
	c, done := readsServer(t)
	defer done()
	ctx := testCtx()
	addr := common.HexToAddress("0x1111111111111111111111111111111111111111")
	const cid int64 = 8453

	checks := []struct {
		name string
		fn   func() error
	}{
		{"IsPaused", func() error { _, e := c.IsPaused(ctx, cid); return e }},
		{"GetFeeBpsOf", func() error { _, e := c.GetFeeBpsOf(ctx, addr, cid); return e }},
		{"HasRules", func() error { _, e := c.HasRules(ctx, cid); return e }},
		{"GetTreasuryAddress", func() error { _, e := c.GetTreasuryAddress(ctx, cid); return e }},
		{"GetRegistryAddress", func() error { _, e := c.GetRegistryAddress(ctx, cid); return e }},
		{"GetPrimaryOperator", func() error { _, e := c.GetPrimaryOperator(ctx, cid); return e }},
		{"IsAfiOperator", func() error { _, e := c.IsAfiOperator(ctx, addr, cid); return e }},
		{"GetOwner", func() error { _, e := c.GetOwner(ctx, addr, cid); return e }},
		{"GetPendingOwner", func() error { _, e := c.GetPendingOwner(ctx, addr, cid); return e }},
		{"GetRoute", func() error { _, e := c.GetRoute(ctx, 1, cid); return e }},
		{"ListRoutes", func() error { _, e := c.ListRoutes(ctx, cid); return e }},
		{"GetTreasuryBalance", func() error { _, e := c.GetTreasuryBalance(ctx, addr, cid); return e }},
	}
	for _, ch := range checks {
		if err := ch.fn(); err != nil {
			t.Errorf("%s: %v", ch.name, err)
		}
	}
}

func TestReads_UnknownChainErrors(t *testing.T) {
	c, done := readsServer(t)
	defer done()
	if _, err := c.IsPaused(testCtx(), 999999); err == nil {
		t.Error("expected error for unknown chain id")
	}
}
