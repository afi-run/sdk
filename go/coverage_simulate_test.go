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
