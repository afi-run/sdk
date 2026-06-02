package afi

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
)

var wethBase = common.HexToAddress("0x4200000000000000000000000000000000000006")

func TestSimulateRoute_Success(t *testing.T) {
	amountOut := big.NewInt(12345)
	srv := newRPCServer(t, rpcHandlers{
		chainID: 8453,
		ethCall: func(common.Address, []byte) []byte {
			return append(encodeUint256(wethBase.Big()), encodeUint256(amountOut)...)
		},
	})
	defer srv.Close()
	rc, err := rpc.DialContext(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer rc.Close()

	res, err := SimulateRoute(context.Background(), SimulateOpts{
		RPCClient:    rc,
		ChainID:      8453,
		QuoterAddr:   common.HexToAddress("0x1111111111111111111111111111111111111111"),
		Asset:        wethBase,
		Amount:       big.NewInt(1_000_000_000_000_000_000),
		StepsEncoded: []byte{0x00},
	})
	if err != nil {
		t.Fatalf("SimulateRoute: %v", err)
	}
	if res.Reverted {
		t.Fatalf("unexpected revert: %x", res.RevertData)
	}
	if res.AmountOut.Cmp(amountOut) != 0 {
		t.Errorf("amountOut = %s, want %s", res.AmountOut, amountOut)
	}
}

func TestSimulateRoute_AutoDetectsUnknownToken(t *testing.T) {
	// A token NOT in the static slot table: SimulateRoute should brute-force
	// detect its balance slot on-chain, cache it, and proceed.
	unknown := common.HexToAddress("0x000000000000000000000000000000000000AbCd")
	defer delete(balanceSlots[8453], unknown)

	amountOut := big.NewInt(999)
	srv := newRPCServer(t, rpcHandlers{
		chainID: 8453,
		ethCall: func(to common.Address, data []byte) []byte {
			// balanceOf(address) probe during detection → echo the sentinel so
			// the first probed slot (0) matches.
			if len(data) >= 4 && data[0] == 0x70 && data[1] == 0xa0 && data[2] == 0x82 && data[3] == 0x31 {
				return encodeUint256(new(big.Int).SetUint64(0xdeadbeefdeadbeef))
			}
			// quote(...) → (outputAsset, amountOut)
			return append(encodeUint256(unknown.Big()), encodeUint256(amountOut)...)
		},
	})
	defer srv.Close()
	rc, err := rpc.DialContext(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer rc.Close()

	if _, ok := LookupBalanceSlot(8453, unknown); ok {
		t.Fatal("precondition: token must start unknown")
	}

	res, err := SimulateRoute(context.Background(), SimulateOpts{
		RPCClient:    rc,
		ChainID:      8453,
		QuoterAddr:   common.HexToAddress("0x1111111111111111111111111111111111111111"),
		Asset:        unknown,
		Amount:       big.NewInt(1_000_000),
		StepsEncoded: []byte{0x00},
	})
	if err != nil {
		t.Fatalf("SimulateRoute: %v", err)
	}
	if res.Reverted || res.AmountOut.Cmp(amountOut) != 0 {
		t.Fatalf("unexpected result: reverted=%v amountOut=%v", res.Reverted, res.AmountOut)
	}
	if slot, ok := LookupBalanceSlot(8453, unknown); !ok || slot != 0 {
		t.Errorf("expected detected slot 0 to be cached, got %d ok=%v", slot, ok)
	}
}

func TestSimulateRoute_Guards(t *testing.T) {
	if _, err := SimulateRoute(context.Background(), SimulateOpts{}); err == nil {
		t.Error("expected error for nil RPCClient")
	}
	rc, err := rpc.DialContext(context.Background(), "http://127.0.0.1:1")
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer rc.Close()
	if _, err := SimulateRoute(context.Background(), SimulateOpts{RPCClient: rc, Amount: big.NewInt(0)}); err == nil {
		t.Error("expected error for non-positive Amount")
	}
}

func TestSimulateRoute_RevertPath(t *testing.T) {
	// Unreachable RPC → CallContext errors → Reverted=true (covers extractRevertData).
	rc, err := rpc.DialContext(context.Background(), "http://127.0.0.1:1")
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer rc.Close()
	res, err := SimulateRoute(context.Background(), SimulateOpts{
		RPCClient:    rc,
		ChainID:      8453,
		QuoterAddr:   common.HexToAddress("0x1111111111111111111111111111111111111111"),
		Asset:        wethBase,
		Amount:       big.NewInt(1_000_000_000_000_000_000),
		StepsEncoded: []byte{0x00},
	})
	if err != nil {
		t.Fatalf("SimulateRoute: %v", err)
	}
	if !res.Reverted {
		t.Error("expected Reverted=true on transport error")
	}
}
