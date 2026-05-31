package afi

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func assertSelector(t *testing.T, data []byte, method string) {
	t.Helper()
	if len(data) < 4 {
		t.Fatalf("%s: calldata too short: %d", method, len(data))
	}
	m, ok := afiParsedABI.Methods[method]
	if !ok {
		t.Fatalf("%s: method not in ABI", method)
	}
	if string(data[:4]) != string(m.ID) {
		t.Errorf("%s: selector mismatch", method)
	}
}

func TestEncodeAfiPause(t *testing.T) {
	data, err := EncodeAfiPause()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 4 {
		t.Fatalf("pause: expected 4 bytes (selector only), got %d", len(data))
	}
	assertSelector(t, data, "pause")
}

func TestEncodeAfiUnpause(t *testing.T) {
	data, err := EncodeAfiUnpause()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 4 {
		t.Fatalf("unpause: expected 4 bytes, got %d", len(data))
	}
	assertSelector(t, data, "unpause")
}

func TestEncodeAfiSetTreasury(t *testing.T) {
	data, err := EncodeAfiSetTreasury(common.HexToAddress("0x1111111111111111111111111111111111111111"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// selector (4) + address padded to 32 = 36
	if len(data) != 4+32 {
		t.Fatalf("setTreasury: expected 36 bytes, got %d", len(data))
	}
	assertSelector(t, data, "setTreasury")
}

func TestEncodeAfiSetFeeBps(t *testing.T) {
	data, err := EncodeAfiSetFeeBps(25)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 4+32 {
		t.Fatalf("setFeeBps: expected 36 bytes, got %d", len(data))
	}
	assertSelector(t, data, "setFeeBps")
}

func TestEncodeAfiSetFeeBps_TooHigh(t *testing.T) {
	if _, err := EncodeAfiSetFeeBps(51); err == nil {
		t.Error("expected error for feeBps > MaxFeeBps, got nil")
	}
}

func TestEncodeAfiSetUserFeeBps(t *testing.T) {
	data, err := EncodeAfiSetUserFeeBps(
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
		40,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 4+32+32 {
		t.Fatalf("setUserFeeBps: expected 68 bytes, got %d", len(data))
	}
	assertSelector(t, data, "setUserFeeBps")
}

func TestEncodeAfiSetUserFeeBps_TooHigh(t *testing.T) {
	if _, err := EncodeAfiSetUserFeeBps(common.HexToAddress("0x2222222222222222222222222222222222222222"), 60); err == nil {
		t.Error("expected error for feeBps > MaxFeeBps, got nil")
	}
}

func TestEncodeAfiSetUserFeeBpsBatch_LengthMismatch(t *testing.T) {
	users := []common.Address{common.HexToAddress("0xaa"), common.HexToAddress("0xbb")}
	if _, err := EncodeAfiSetUserFeeBpsBatch(users, []uint16{1}); err == nil {
		t.Error("expected error for length mismatch, got nil")
	}
}

func TestEncodeAfiSetTreasury_ZeroAddress(t *testing.T) {
	if _, err := EncodeAfiSetTreasury(common.Address{}); err == nil {
		t.Error("expected error for zero address, got nil")
	}
}

func TestEncodeAfiAddRule_ZeroAddress(t *testing.T) {
	if _, err := EncodeAfiAddRule(common.Address{}); err == nil {
		t.Error("expected error for zero address, got nil")
	}
}

func TestEncodeAfiSetUserFeeBpsBatch(t *testing.T) {
	users := []common.Address{
		common.HexToAddress("0x3333333333333333333333333333333333333333"),
		common.HexToAddress("0x4444444444444444444444444444444444444444"),
	}
	feesBps := []uint16{10, 20}
	data, err := EncodeAfiSetUserFeeBpsBatch(users, feesBps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertSelector(t, data, "setUserFeeBpsBatch")
}

func TestEncodeAfiClearUserFeeBps(t *testing.T) {
	data, err := EncodeAfiClearUserFeeBps(common.HexToAddress("0x5555555555555555555555555555555555555555"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 4+32 {
		t.Fatalf("clearUserFeeBps: expected 36 bytes, got %d", len(data))
	}
	assertSelector(t, data, "clearUserFeeBps")
}

func TestEncodeAfiResetAnyUserOverride(t *testing.T) {
	data, err := EncodeAfiResetAnyUserOverride()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 4 {
		t.Fatalf("resetAnyUserOverride: expected 4 bytes, got %d", len(data))
	}
	assertSelector(t, data, "resetAnyUserOverride")
}

func TestEncodeAfiAddRule(t *testing.T) {
	data, err := EncodeAfiAddRule(common.HexToAddress("0x6666666666666666666666666666666666666666"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 4+32 {
		t.Fatalf("addRule: expected 36 bytes, got %d", len(data))
	}
	assertSelector(t, data, "addRule")
}

func TestEncodeAfiClearRules(t *testing.T) {
	data, err := EncodeAfiClearRules()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 4 {
		t.Fatalf("clearRules: expected 4 bytes, got %d", len(data))
	}
	assertSelector(t, data, "clearRules")
}

func TestEncodeAfiSetOperator(t *testing.T) {
	data, err := EncodeAfiSetOperator(
		common.HexToAddress("0x7777777777777777777777777777777777777777"),
		true,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 4+32+32 {
		t.Fatalf("setOperator: expected 68 bytes, got %d", len(data))
	}
	assertSelector(t, data, "setOperator")
}

func TestEncodeAfiRescueTokens(t *testing.T) {
	data, err := EncodeAfiRescueTokens(
		common.HexToAddress("0x8888888888888888888888888888888888888888"),
		big.NewInt(1_000_000),
		common.HexToAddress("0x9999999999999999999999999999999999999999"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 4+32+32+32 {
		t.Fatalf("rescueTokens: expected 100 bytes, got %d", len(data))
	}
	assertSelector(t, data, "rescueTokens")
}
