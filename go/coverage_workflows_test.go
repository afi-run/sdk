package afi

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// TestWorkflows_WriteWrappers exercises every admin/swap wrapper through the
// fake RPC write path (estimateGas → sendRawTransaction → receipt). It covers
// the wrappers plus afiAddressForCtx / SendContractTx.
func TestWorkflows_WriteWrappers(t *testing.T) {
	srv := newRPCServer(t, rpcHandlers{
		chainID:     8453,
		nonce:       1,
		gasEstimate: 300_000,
		baseFee:     big.NewInt(1_000_000_000),
		gasTip:      big.NewInt(1_000_000),
		// Large value so the SwapFor allowance precheck passes.
		ethCall:         func(common.Address, []byte) []byte { return encodeUint256(new(big.Int).Lsh(big.NewInt(1), 200)) },
		receiptSucceeds: true,
	})
	defer srv.Close()
	c := mustClientWith(t, srv.URL, testPrivKey)
	defer c.Close()

	ctx := testCtx()
	a := common.HexToAddress("0x1111111111111111111111111111111111111111")
	amt := big.NewInt(1000)
	params := []byte{0x00}

	calls := []struct {
		name string
		fn   func() (*types.Receipt, error)
	}{
		{"SwapFor", func() (*types.Receipt, error) {
			return c.SwapFor(ctx, a, a, amt, a, big.NewInt(0), params, WithoutAllowancePrecheck())
		}},
		{"BatchSwapFor", func() (*types.Receipt, error) {
			return c.BatchSwapFor(ctx, []SwapForRequest{{User: a, TokenIn: a, AmountIn: amt, TokenOut: a, MinOut: big.NewInt(0), Params: params}}, WithoutAllowancePrecheck())
		}},
		{"AdminPause", func() (*types.Receipt, error) { return c.AdminPause(ctx) }},
		{"AdminUnpause", func() (*types.Receipt, error) { return c.AdminUnpause(ctx) }},
		{"AdminSetTreasury", func() (*types.Receipt, error) { return c.AdminSetTreasury(ctx, a) }},
		{"AdminSetFeeBps", func() (*types.Receipt, error) { return c.AdminSetFeeBps(ctx, 30) }},
		{"AdminSetUserFeeBps", func() (*types.Receipt, error) { return c.AdminSetUserFeeBps(ctx, a, 30) }},
		{"AdminSetUserFeeBpsBatch", func() (*types.Receipt, error) {
			return c.AdminSetUserFeeBpsBatch(ctx, []common.Address{a}, []uint16{30})
		}},
		{"AdminClearUserFeeBps", func() (*types.Receipt, error) { return c.AdminClearUserFeeBps(ctx, a) }},
		{"AdminResetAnyUserOverride", func() (*types.Receipt, error) { return c.AdminResetAnyUserOverride(ctx) }},
		{"AdminAddRule", func() (*types.Receipt, error) { return c.AdminAddRule(ctx, a) }},
		{"AdminClearRules", func() (*types.Receipt, error) { return c.AdminClearRules(ctx) }},
		{"AdminSetOperator", func() (*types.Receipt, error) { return c.AdminSetOperator(ctx, a, true) }},
		{"AdminRescueTokens", func() (*types.Receipt, error) { return c.AdminRescueTokens(ctx, a, amt, a) }},
	}
	for _, cl := range calls {
		if _, err := cl.fn(); err != nil {
			t.Errorf("%s: %v", cl.name, err)
		}
	}
}
