package afi

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestMatchesAndDecode_UnknownEvent(t *testing.T) {
	log := &types.Log{Topics: []common.Hash{common.HexToHash("0x01")}}
	if matchesEvent(afiParsedABI, "DoesNotExist", log) {
		t.Error("matchesEvent should be false for an event not in the ABI")
	}
	var out struct{}
	if err := decodeEventLog(afiParsedABI, "DoesNotExist", log, &out); err == nil {
		t.Error("decodeEventLog should error for an event not in the ABI")
	}
}

func TestAddressURL_AndTxURL(t *testing.T) {
	addr := "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
	if u, err := AddressURL(addr, NetworkBase); err != nil || u == "" {
		t.Errorf("AddressURL(base): %q %v", u, err)
	}
	if _, err := AddressURL(addr, Network("nope")); err == nil {
		t.Error("AddressURL: expected error for unknown network")
	}

	c := &Client{}
	if u, err := c.AddressURL(addr, NetworkBase); err != nil || u == "" {
		t.Errorf("client.AddressURL: %q %v", u, err)
	}
	if u, err := c.TxURL("0x"+common.Bytes2Hex(make([]byte, 32)), NetworkBase); err != nil || u == "" {
		t.Errorf("client.TxURL: %q %v", u, err)
	}
}

func TestBalanceSlotRegistry(t *testing.T) {
	tok := common.HexToAddress("0x9999999999999999999999999999999999999999")
	if _, ok := LookupBalanceSlot(8453, tok); ok {
		t.Error("expected unknown slot before registration")
	}
	RegisterBalanceSlot(8453, tok, 7)
	if slot, ok := LookupBalanceSlot(8453, tok); !ok || slot != 7 {
		t.Errorf("after register: slot=%d ok=%v, want 7 true", slot, ok)
	}
}

func TestBatchLengthMismatch(t *testing.T) {
	// EncodeAfiSetUserFeeBpsBatch is a pure encoder — no RPC needed.
	users := []common.Address{common.HexToAddress("0x01")}
	if _, err := EncodeAfiSetUserFeeBpsBatch(users, []uint16{1, 2}); err == nil {
		t.Error("EncodeAfiSetUserFeeBpsBatch: expected length-mismatch error")
	}
	if _, err := EncodeAfiSetUserFeeBpsBatch(users, []uint16{1}); err != nil {
		t.Errorf("EncodeAfiSetUserFeeBpsBatch valid: %v", err)
	}

	// AdminSetUserFeeBpsBatch validates lengths before touching the RPC.
	c := mustClientWith(t, "http://127.0.0.1:1", testPrivKey)
	defer c.Close()
	if _, err := c.AdminSetUserFeeBpsBatch(testCtx(), users, []uint16{1, 2}); err == nil {
		t.Error("AdminSetUserFeeBpsBatch: expected length-mismatch error")
	}
}

func TestSwapFor_PrecheckPaths(t *testing.T) {
	// Sufficient allowance → precheck passes, tx is sent.
	okSrv := newRPCServer(t, rpcHandlers{
		chainID: 8453, nonce: 1, gasEstimate: 300_000,
		baseFee: big.NewInt(1_000_000_000), gasTip: big.NewInt(1_000_000),
		ethCall:         func(common.Address, []byte) []byte { return encodeUint256(new(big.Int).Lsh(big.NewInt(1), 200)) },
		receiptSucceeds: true,
	})
	defer okSrv.Close()
	c := mustClientWith(t, okSrv.URL, testPrivKey)
	defer c.Close()
	a := common.HexToAddress("0x1111111111111111111111111111111111111111")
	if _, err := c.SwapFor(testCtx(), a, a, big.NewInt(1000), a, big.NewInt(0), []byte{0x00}); err != nil {
		t.Errorf("SwapFor with sufficient allowance: %v", err)
	}

	// Insufficient allowance → precheck fails before sending.
	lowSrv := newRPCServer(t, rpcHandlers{
		chainID: 8453,
		ethCall: func(common.Address, []byte) []byte { return encodeUint256(big.NewInt(1)) },
	})
	defer lowSrv.Close()
	c2 := mustClientWith(t, lowSrv.URL, testPrivKey)
	defer c2.Close()
	if _, err := c2.SwapFor(testCtx(), a, a, big.NewInt(1000), a, big.NewInt(0), []byte{0x00}); err == nil {
		t.Error("SwapFor with short allowance: expected precheck error")
	}
}
