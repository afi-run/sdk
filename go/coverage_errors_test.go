package afi

import (
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// badLog builds a log whose topic0 matches the named event (so matchesEvent
// passes) but whose payload is malformed (1 byte of data, no indexed topics),
// forcing decodeEventLog to error — exercising every parser's error branch.
func badLog(parsed abi.ABI, name string) *types.Log {
	return &types.Log{
		Topics: []common.Hash{parsed.Events[name].ID},
		Data:   []byte{0x01},
	}
}

func TestEventParsers_DecodeErrors(t *testing.T) {
	afi := func(n string) []*types.Log { return []*types.Log{badLog(afiParsedABI, n)} }
	nmr := func(n string) []*types.Log { return []*types.Log{badLog(nmrParsedABI, n)} }

	cases := []struct {
		name string
		err  error
	}{
		{"ParseSwapExecuted", func() error { _, e := ParseSwapExecuted(afi("SwapExecuted")); return e }()},
		{"ParseFeeCollected", func() error { _, e := ParseFeeCollected(afi("FeeCollected")); return e }()},
		{"ParseAfiTreasuryUpdated", func() error { _, e := ParseAfiTreasuryUpdated(afi("TreasuryUpdated")); return e }()},
		{"ParseNMRTreasuryUpdated", func() error { _, e := ParseNMRTreasuryUpdated(nmr("TreasuryUpdated")); return e }()},
		{"ParseFeeBpsUpdated", func() error { _, e := ParseFeeBpsUpdated(afi("FeeBpsUpdated")); return e }()},
		{"ParseUserFeeBpsSet", func() error { _, e := ParseUserFeeBpsSet(afi("UserFeeBpsSet")); return e }()},
		{"ParseUserFeeBpsCleared", func() error { _, e := ParseUserFeeBpsCleared(afi("UserFeeBpsCleared")); return e }()},
		{"ParseFlashLoanRequested", func() error { _, e := ParseFlashLoanRequested(nmr("FlashLoanRequested")); return e }()},
		{"ParseFlashLoanExecuted", func() error { _, e := ParseFlashLoanExecuted(nmr("FlashLoanExecuted")); return e }()},
		{"ParseFlashLoanFailed", func() error { _, e := ParseFlashLoanFailed(nmr("FlashLoanFailed")); return e }()},
		{"ParseFlashLoanFailedWithData", func() error { _, e := ParseFlashLoanFailedWithData(nmr("FlashLoanFailedWithData")); return e }()},
		{"ParseNMRSwapExecuted", func() error { _, e := ParseNMRSwapExecuted(nmr("SwapExecuted")); return e }()},
		{"ParseProfitSwept", func() error { _, e := ParseProfitSwept(nmr("ProfitSwept")); return e }()},
		{"ParseProfitShareUpdated", func() error { _, e := ParseProfitShareUpdated(nmr("ProfitShareUpdated")); return e }()},
	}
	for _, c := range cases {
		if c.err == nil {
			t.Errorf("%s: expected decode error, got nil", c.name)
		}
	}
}

// TestEventParsers_SkipNonMatching feeds each parser a log whose topic0 matches
// nothing, exercising the matchesEvent=false `continue` branch and the empty
// `return out, nil` path.
func TestEventParsers_SkipNonMatching(t *testing.T) {
	noise := []*types.Log{{Topics: []common.Hash{common.HexToHash("0xdeadbeef")}}}
	results := []int{}
	add := func(n int, err error) {
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		results = append(results, n)
	}
	r1, e := ParseSwapExecuted(noise)
	add(len(r1), e)
	r2, e := ParseFeeCollected(noise)
	add(len(r2), e)
	r3, e := ParseAfiTreasuryUpdated(noise)
	add(len(r3), e)
	r4, e := ParseNMRTreasuryUpdated(noise)
	add(len(r4), e)
	r5, e := ParseFeeBpsUpdated(noise)
	add(len(r5), e)
	r6, e := ParseUserFeeBpsSet(noise)
	add(len(r6), e)
	r7, e := ParseUserFeeBpsCleared(noise)
	add(len(r7), e)
	r8, e := ParseFlashLoanRequested(noise)
	add(len(r8), e)
	r9, e := ParseFlashLoanExecuted(noise)
	add(len(r9), e)
	r10, e := ParseFlashLoanFailed(noise)
	add(len(r10), e)
	r11, e := ParseFlashLoanFailedWithData(noise)
	add(len(r11), e)
	r12, e := ParseNMRSwapExecuted(noise)
	add(len(r12), e)
	r13, e := ParseProfitSwept(noise)
	add(len(r13), e)
	r14, e := ParseProfitShareUpdated(noise)
	add(len(r14), e)
	for i, n := range results {
		if n != 0 {
			t.Errorf("parser #%d returned %d events for non-matching log, want 0", i, n)
		}
	}
}

func TestReads_DecodeErrors(t *testing.T) {
	// eth_call returns 1 byte — too short to unpack into bool/address/uint, so
	// every read hits its decode-error branch.
	srv := newRPCServer(t, rpcHandlers{
		chainID: 8453,
		ethCall: func(common.Address, []byte) []byte { return []byte{0x01} },
	})
	defer srv.Close()
	c := mustClientWith(t, srv.URL, testPrivKey)
	defer c.Close()
	ctx := testCtx()
	a := common.HexToAddress("0x1111111111111111111111111111111111111111")
	const cid int64 = 8453

	checks := []struct {
		name string
		err  error
	}{
		{"IsPaused", func() error { _, e := c.IsPaused(ctx, cid); return e }()},
		{"GetFeeBpsOf", func() error { _, e := c.GetFeeBpsOf(ctx, a, cid); return e }()},
		{"HasRules", func() error { _, e := c.HasRules(ctx, cid); return e }()},
		{"GetTreasuryAddress", func() error { _, e := c.GetTreasuryAddress(ctx, cid); return e }()},
		{"GetRegistryAddress", func() error { _, e := c.GetRegistryAddress(ctx, cid); return e }()},
		{"GetPrimaryOperator", func() error { _, e := c.GetPrimaryOperator(ctx, cid); return e }()},
		{"IsAfiOperator", func() error { _, e := c.IsAfiOperator(ctx, a, cid); return e }()},
		{"GetOwner", func() error { _, e := c.GetOwner(ctx, a, cid); return e }()},
		{"GetPendingOwner", func() error { _, e := c.GetPendingOwner(ctx, a, cid); return e }()},
		{"GetNMRTreasury", func() error { _, e := c.GetNMRTreasury(ctx, cid); return e }()},
		{"GetNMRProfitShare", func() error { _, e := c.GetNMRProfitShare(ctx, cid); return e }()},
		{"IsNMROperator", func() error { _, e := c.IsNMROperator(ctx, a, cid); return e }()},
		{"GetRoute", func() error { _, e := c.GetRoute(ctx, 1, cid); return e }()},
		{"GetTreasuryBalance", func() error { _, e := c.GetTreasuryBalance(ctx, a, cid); return e }()},
	}
	for _, ch := range checks {
		if ch.err == nil {
			t.Errorf("%s: expected decode error, got nil", ch.name)
		}
	}
}

func TestWorkflows_AddressResolutionErrors(t *testing.T) {
	// Unknown chain id → afiAddressForCtx / nmrAddressForCtx fail, so every
	// wrapper returns early at its address-resolution error branch.
	srv := newRPCServer(t, rpcHandlers{chainID: 999999})
	defer srv.Close()
	c := mustClientWith(t, srv.URL, testPrivKey)
	defer c.Close()
	ctx := testCtx()
	a := common.HexToAddress("0x1111111111111111111111111111111111111111")
	amt := big.NewInt(1000)
	params := []byte{0x00}

	checks := []struct {
		name string
		err  error
	}{
		{"NMRArbitrage", func() error { _, e := c.NMRArbitrage(ctx, a, amt, params); return e }()},
		{"NMRCycleSwap", func() error { _, e := c.NMRCycleSwap(ctx, a, amt, big.NewInt(0), params); return e }()},
		{"NMRLoanArbitrage", func() error {
			_, e := c.NMRLoanArbitrage(ctx, a, a, amt, big.NewInt(0), params, WithoutAllowancePrecheck())
			return e
		}()},
		{"SweepNMRProfit", func() error { _, e := c.SweepNMRProfit(ctx, a, amt); return e }()},
		{"SwapFor", func() error {
			_, e := c.SwapFor(ctx, a, a, amt, a, big.NewInt(0), params, WithoutAllowancePrecheck())
			return e
		}()},
		{"BatchSwapFor", func() error { _, e := c.BatchSwapFor(ctx, nil, WithoutAllowancePrecheck()); return e }()},
		{"AdminPause", func() error { _, e := c.AdminPause(ctx); return e }()},
		{"AdminUnpause", func() error { _, e := c.AdminUnpause(ctx); return e }()},
		{"AdminSetTreasury", func() error { _, e := c.AdminSetTreasury(ctx, a); return e }()},
		{"AdminSetFeeBps", func() error { _, e := c.AdminSetFeeBps(ctx, 30); return e }()},
		{"AdminSetUserFeeBps", func() error { _, e := c.AdminSetUserFeeBps(ctx, a, 30); return e }()},
		{"AdminSetUserFeeBpsBatch", func() error { _, e := c.AdminSetUserFeeBpsBatch(ctx, []common.Address{a}, []uint16{30}); return e }()},
		{"AdminClearUserFeeBps", func() error { _, e := c.AdminClearUserFeeBps(ctx, a); return e }()},
		{"AdminResetAnyUserOverride", func() error { _, e := c.AdminResetAnyUserOverride(ctx); return e }()},
		{"AdminAddRule", func() error { _, e := c.AdminAddRule(ctx, a); return e }()},
		{"AdminClearRules", func() error { _, e := c.AdminClearRules(ctx); return e }()},
		{"AdminSetOperator", func() error { _, e := c.AdminSetOperator(ctx, a, true); return e }()},
		{"AdminRescueTokens", func() error { _, e := c.AdminRescueTokens(ctx, a, amt, a); return e }()},
	}
	for _, ch := range checks {
		if ch.err == nil {
			t.Errorf("%s: expected address-resolution error, got nil", ch.name)
		}
	}
}

// TestRPCUnreachable_TransportErrors points the client at a dead port so every
// eth_* call fails at dial — covering the callRead error branch in reads and the
// DetectNetwork error branch in afiAddressForCtx / nmrAddressForCtx.
func TestRPCUnreachable_TransportErrors(t *testing.T) {
	c := mustClientWith(t, "http://127.0.0.1:1", testPrivKey)
	defer c.Close()
	ctx := testCtx()
	a := common.HexToAddress("0x1111111111111111111111111111111111111111")
	amt := big.NewInt(1000)
	params := []byte{0x00}
	const cid int64 = 8453

	reads := []func() error{
		func() error { _, e := c.IsPaused(ctx, cid); return e },
		func() error { _, e := c.GetFeeBpsOf(ctx, a, cid); return e },
		func() error { _, e := c.HasRules(ctx, cid); return e },
		func() error { _, e := c.GetTreasuryAddress(ctx, cid); return e },
		func() error { _, e := c.GetRegistryAddress(ctx, cid); return e },
		func() error { _, e := c.GetPrimaryOperator(ctx, cid); return e },
		func() error { _, e := c.IsAfiOperator(ctx, a, cid); return e },
		func() error { _, e := c.GetOwner(ctx, a, cid); return e },
		func() error { _, e := c.GetPendingOwner(ctx, a, cid); return e },
		func() error { _, e := c.GetNMRTreasury(ctx, cid); return e },
		func() error { _, e := c.GetNMRProfitShare(ctx, cid); return e },
		func() error { _, e := c.IsNMROperator(ctx, a, cid); return e },
		func() error { _, e := c.GetRoute(ctx, 1, cid); return e },
		func() error { _, e := c.ListRoutes(ctx, cid); return e },
		func() error { _, e := c.GetTreasuryBalance(ctx, a, cid); return e },
		func() error { _, e := c.VerifyDeployment(ctx, cid); return e },
	}
	for i, fn := range reads {
		if err := fn(); err == nil {
			t.Errorf("read #%d: expected transport error, got nil", i)
		}
	}

	writes := []func() error{
		func() error { _, e := c.NMRArbitrage(ctx, a, amt, params); return e },
		func() error { _, e := c.NMRCycleSwap(ctx, a, amt, big.NewInt(0), params); return e },
		func() error {
			_, e := c.NMRLoanArbitrage(ctx, a, a, amt, big.NewInt(0), params, WithoutAllowancePrecheck())
			return e
		},
		func() error { _, e := c.SweepNMRProfit(ctx, a, amt); return e },
		func() error { _, e := c.AdminPause(ctx); return e },
		func() error { _, e := c.AdminRescueTokens(ctx, a, amt, a); return e },
	}
	for i, fn := range writes {
		if err := fn(); err == nil {
			t.Errorf("write #%d: expected transport error, got nil", i)
		}
	}
}

func TestQuoterEndpoints_ErrorPaths(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := &Client{apiURL: srv.URL}
	ctx := testCtx()

	checks := []struct {
		name string
		err  error
	}{
		{"FindArbitrage", func() error { _, e := c.FindArbitrage(ctx, ArbitrageRequest{}); return e }()},
		{"FindPath", func() error { _, e := c.FindPath(ctx, PathRequest{}); return e }()},
		{"GetRoutes", func() error { _, e := c.GetRoutes(ctx, RoutesRequest{}); return e }()},
		{"PriceQuote", func() error { _, e := c.PriceQuote(ctx, PriceQuoteRequest{}); return e }()},
		{"QuoteDex", func() error { _, e := c.QuoteDex(ctx, "uniV3", DexQuoteRequest{}); return e }()},
		{"GetLiquidationCandidates", func() error { _, e := c.GetLiquidationCandidates(ctx, LiquidationCandidatesRequest{}); return e }()},
		{"Liquidate", func() error { _, e := c.Liquidate(ctx, LiquidateRequest{}); return e }()},
	}
	for _, ch := range checks {
		if ch.err == nil {
			t.Errorf("%s: expected error on HTTP 500, got nil", ch.name)
		}
	}
}
