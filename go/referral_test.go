package afi

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func assertReferralSelector(t *testing.T, data []byte, method string) {
	t.Helper()
	if len(data) < 4 {
		t.Fatalf("%s: calldata too short: %d", method, len(data))
	}
	m, ok := referralRouterParsedABI.Methods[method]
	if !ok {
		t.Fatalf("%s: method not in ABI", method)
	}
	if string(data[:4]) != string(m.ID) {
		t.Errorf("%s: selector mismatch", method)
	}
}

var (
	refTokenIn  = common.HexToAddress("0x1111111111111111111111111111111111111111")
	refTokenOut = common.HexToAddress("0x2222222222222222222222222222222222222222")
	refReferrer = common.HexToAddress("0x3333333333333333333333333333333333333333")
	refUser     = common.HexToAddress("0x4444444444444444444444444444444444444444")
	refDelegate = common.HexToAddress("0x5555555555555555555555555555555555555555")
	refTo       = common.HexToAddress("0x6666666666666666666666666666666666666666")
)

func TestReferralRouterAddress(t *testing.T) {
	addr, net, err := ReferralRouterAddress(8453)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if net != NetworkBase {
		t.Errorf("expected NetworkBase, got %s", net)
	}
	if addr != ReferralRouterAddresses[NetworkBase] {
		t.Errorf("address mismatch: %s", addr)
	}
}

func TestReferralRouterAddress_UnknownChain(t *testing.T) {
	if _, _, err := ReferralRouterAddress(999999); err == nil {
		t.Error("expected error for unknown chain id, got nil")
	}
}

func TestEncodeSwapWithReferral(t *testing.T) {
	data, err := EncodeSwapWithReferral(refTokenIn, big.NewInt(1000), refTokenOut, big.NewInt(900), []byte{0x01, 0x02}, refReferrer, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertReferralSelector(t, data, "swapWithReferral")

	// Round-trip the args to make sure they encode in the right order.
	args, err := referralRouterParsedABI.Methods["swapWithReferral"].Inputs.Unpack(data[4:])
	if err != nil {
		t.Fatalf("unpack: %v", err)
	}
	if args[0].(common.Address) != refTokenIn {
		t.Errorf("tokenIn mismatch: %v", args[0])
	}
	if args[1].(*big.Int).Cmp(big.NewInt(1000)) != 0 {
		t.Errorf("amountIn mismatch: %v", args[1])
	}
	if args[6].(uint16) != 5 {
		t.Errorf("referralBps mismatch: %v", args[6])
	}
}

func TestEncodeSwapWithReferral_ZeroAmount(t *testing.T) {
	if _, err := EncodeSwapWithReferral(refTokenIn, big.NewInt(0), refTokenOut, big.NewInt(0), nil, refReferrer, 5); err == nil {
		t.Error("expected error for zero amountIn, got nil")
	}
	if _, err := EncodeSwapWithReferral(refTokenIn, nil, refTokenOut, big.NewInt(0), nil, refReferrer, 5); err == nil {
		t.Error("expected error for nil amountIn, got nil")
	}
}

func TestEncodeSwapWithReferral_BpsTooHigh(t *testing.T) {
	if _, err := EncodeSwapWithReferral(refTokenIn, big.NewInt(1), refTokenOut, big.NewInt(0), nil, refReferrer, ReferralHardCapBps+1); err == nil {
		t.Error("expected error for referralBps > hard cap, got nil")
	}
}

func TestEncodeSwapWithReferralFor(t *testing.T) {
	data, err := EncodeSwapWithReferralFor(refUser, refTokenIn, big.NewInt(1000), refTokenOut, big.NewInt(900), nil, refReferrer, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertReferralSelector(t, data, "swapWithReferralFor")
}

func TestEncodeSwapWithReferralFor_Errors(t *testing.T) {
	if _, err := EncodeSwapWithReferralFor(common.Address{}, refTokenIn, big.NewInt(1), refTokenOut, big.NewInt(0), nil, refReferrer, 5); err == nil {
		t.Error("expected error for zero user, got nil")
	}
	if _, err := EncodeSwapWithReferralFor(refUser, refTokenIn, big.NewInt(0), refTokenOut, big.NewInt(0), nil, refReferrer, 5); err == nil {
		t.Error("expected error for zero amountIn, got nil")
	}
	if _, err := EncodeSwapWithReferralFor(refUser, refTokenIn, big.NewInt(1), refTokenOut, big.NewInt(0), nil, refReferrer, ReferralHardCapBps+1); err == nil {
		t.Error("expected error for referralBps > hard cap, got nil")
	}
}

func TestEncodeReferralClaim(t *testing.T) {
	data, err := EncodeReferralClaim(refTokenOut, refTo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 4+32+32 {
		t.Fatalf("claim: expected 68 bytes, got %d", len(data))
	}
	assertReferralSelector(t, data, "claim")
}

func TestEncodeReferralClaim_ZeroTo(t *testing.T) {
	if _, err := EncodeReferralClaim(refTokenOut, common.Address{}); err == nil {
		t.Error("expected error for zero to, got nil")
	}
}

func TestEncodeReferralClaimMany(t *testing.T) {
	data, err := EncodeReferralClaimMany([]common.Address{refTokenIn, refTokenOut}, refTo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertReferralSelector(t, data, "claimMany")
}

func TestEncodeReferralClaimMany_Errors(t *testing.T) {
	if _, err := EncodeReferralClaimMany(nil, refTo); err == nil {
		t.Error("expected error for empty tokens, got nil")
	}
	if _, err := EncodeReferralClaimMany([]common.Address{refTokenIn}, common.Address{}); err == nil {
		t.Error("expected error for zero to, got nil")
	}
}

func TestEncodeSetDelegateAllowance(t *testing.T) {
	data, err := EncodeSetDelegateAllowance(refTokenIn, refDelegate, big.NewInt(500), 1893456000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertReferralSelector(t, data, "setDelegateAllowance")
}

func TestEncodeSetDelegateAllowance_Errors(t *testing.T) {
	if _, err := EncodeSetDelegateAllowance(refTokenIn, common.Address{}, big.NewInt(1), 1); err == nil {
		t.Error("expected error for zero delegate, got nil")
	}
	if _, err := EncodeSetDelegateAllowance(refTokenIn, refDelegate, nil, 1); err == nil {
		t.Error("expected error for nil amount, got nil")
	}
	if _, err := EncodeSetDelegateAllowance(refTokenIn, refDelegate, big.NewInt(-1), 1); err == nil {
		t.Error("expected error for negative amount, got nil")
	}
	// amount exceeding uint208
	over208 := new(big.Int).Lsh(big.NewInt(1), 208)
	if _, err := EncodeSetDelegateAllowance(refTokenIn, refDelegate, over208, 1); err == nil {
		t.Error("expected error for amount > uint208, got nil")
	}
	// deadline exceeding uint48
	if _, err := EncodeSetDelegateAllowance(refTokenIn, refDelegate, big.NewInt(1), (1<<48)); err == nil {
		t.Error("expected error for deadline > uint48, got nil")
	}
}

func TestEncodeRevokeDelegate(t *testing.T) {
	data, err := EncodeRevokeDelegate(refTokenIn, refDelegate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertReferralSelector(t, data, "revokeDelegate")
}

func TestEncodeRevokeDelegate_ZeroDelegate(t *testing.T) {
	if _, err := EncodeRevokeDelegate(refTokenIn, common.Address{}); err == nil {
		t.Error("expected error for zero delegate, got nil")
	}
}

func TestEncodeReferralPauseUnpause(t *testing.T) {
	d1, err := EncodeReferralPause()
	if err != nil || len(d1) != 4 {
		t.Fatalf("pause: err=%v len=%d", err, len(d1))
	}
	assertReferralSelector(t, d1, "pause")

	d2, err := EncodeReferralUnpause()
	if err != nil || len(d2) != 4 {
		t.Fatalf("unpause: err=%v len=%d", err, len(d2))
	}
	assertReferralSelector(t, d2, "unpause")
}

func TestEncodeReferralSetMaxReferralBps(t *testing.T) {
	data, err := EncodeReferralSetMaxReferralBps(ReferralHardCapBps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertReferralSelector(t, data, "setMaxReferralBps")

	if _, err := EncodeReferralSetMaxReferralBps(ReferralHardCapBps + 1); err == nil {
		t.Error("expected error for bps > hard cap, got nil")
	}
}

func TestEncodeReferralRescueTokens(t *testing.T) {
	data, err := EncodeReferralRescueTokens(refTokenOut, refTo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertReferralSelector(t, data, "rescueTokens")

	if _, err := EncodeReferralRescueTokens(refTokenOut, common.Address{}); err == nil {
		t.Error("expected error for zero to, got nil")
	}
}
