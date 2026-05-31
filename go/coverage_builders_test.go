package afi

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

var (
	over128 = new(big.Int).Lsh(big.NewInt(1), 128) // 2^128 — out of uint128 range
	over160 = new(big.Int).Lsh(big.NewInt(1), 160) // 2^160 — out of uint160 range
	addrX   = common.HexToAddress("0x2222222222222222222222222222222222222222")
)

// TestBuilders_ValidationErrors drives every step builder with an out-of-range
// minOut (and a few with out-of-range sqrt limits / tick spacings) to exercise
// the validation error branches.
func TestBuilders_ValidationErrors(t *testing.T) {
	checks := []struct {
		name string
		err  error
	}{
		{"UniV3 minOut", func() error { _, e := BuildUniV3Step(addrX, 500, over128, big.NewInt(0)); return e }()},
		{"UniV3 sqrt", func() error { _, e := BuildUniV3Step(addrX, 500, big.NewInt(1), over160); return e }()},
		{"CakeV3 minOut", func() error { _, e := BuildCakeV3Step(addrX, 500, over128, big.NewInt(0)); return e }()},
		{"CakeV3 sqrt", func() error { _, e := BuildCakeV3Step(addrX, 500, big.NewInt(1), over160); return e }()},
		{"UniV4 minOut", func() error { _, e := BuildUniV4Step(addrX, addrX, 500, 10, addrX, true, over128); return e }()},
		{"UniV4 tick", func() error { _, e := BuildUniV4Step(addrX, addrX, 500, 1<<23, addrX, true, big.NewInt(1)); return e }()},
		{"Aerodrome minOut", func() error { _, e := BuildAerodromeStep(addrX, 10, big.NewInt(0), over128); return e }()},
		{"Aerodrome sqrt", func() error { _, e := BuildAerodromeStep(addrX, 10, over160, big.NewInt(1)); return e }()},
		{"Aerodrome tick", func() error { _, e := BuildAerodromeStep(addrX, 1<<23, big.NewInt(0), big.NewInt(1)); return e }()},
		{"BalancerV3 minOut", func() error { _, e := BuildBalancerV3Step(addrX, addrX, over128); return e }()},
		{"Fluid minOut", func() error { _, e := BuildFluidStep(addrX, true, addrX, over128); return e }()},
		{"Curve128 minDy", func() error { _, e := BuildCurve128Step(0, 1, over128, addrX, addrX); return e }()},
		{"Curve256 minDy", func() error { _, e := BuildCurve256Step(0, 1, over160, addrX, addrX); return e }()},
	}
	for _, c := range checks {
		if c.err == nil {
			t.Errorf("%s: expected validation error, got nil", c.name)
		}
	}
}

// TestBuilders_HappyPath covers the success branch of each builder.
func TestBuilders_HappyPath(t *testing.T) {
	mk := []struct {
		name string
		err  error
	}{
		{"UniV3", func() error { _, e := BuildUniV3Step(addrX, 500, big.NewInt(1), big.NewInt(0)); return e }()},
		{"CakeV3", func() error { _, e := BuildCakeV3Step(addrX, 500, big.NewInt(1), big.NewInt(0)); return e }()},
		{"UniV4", func() error { _, e := BuildUniV4Step(addrX, addrX, 500, 10, addrX, true, big.NewInt(1)); return e }()},
		{"Aerodrome", func() error { _, e := BuildAerodromeStep(addrX, 10, big.NewInt(0), big.NewInt(1)); return e }()},
		{"BalancerV3", func() error { _, e := BuildBalancerV3Step(addrX, addrX, big.NewInt(1)); return e }()},
		{"Fluid", func() error { _, e := BuildFluidStep(addrX, true, addrX, big.NewInt(1)); return e }()},
		{"Curve128", func() error { _, e := BuildCurve128Step(0, 1, big.NewInt(1), addrX, addrX); return e }()},
		{"Curve256", func() error { _, e := BuildCurve256Step(0, 1, big.NewInt(1), addrX, addrX); return e }()},
		{"AaveLiquidator", func() error { _, e := BuildAaveLiquidatorStep(addrX, addrX, addrX); return e }()},
	}
	for _, c := range mk {
		if c.err != nil {
			t.Errorf("%s: unexpected error: %v", c.name, c.err)
		}
	}
}

func TestParseBigInt(t *testing.T) {
	if b, err := parseBigInt("12345"); err != nil || b.Int64() != 12345 {
		t.Errorf("parseBigInt(12345): %v %v", b, err)
	}
	if _, err := parseBigInt("not-a-number"); err == nil {
		t.Error("parseBigInt: expected error for non-numeric")
	}
}

func TestUnmarshalJSON_Errors(t *testing.T) {
	var q Quote
	if err := q.UnmarshalJSON([]byte("not json")); err == nil {
		t.Error("Quote.UnmarshalJSON: expected error on malformed JSON")
	}
	if err := q.UnmarshalJSON([]byte(`{"amountInWei":"xx"}`)); err == nil {
		t.Error("Quote.UnmarshalJSON: expected error on bad bigint")
	}
	var r SwapResult
	if err := r.UnmarshalJSON([]byte("not json")); err == nil {
		t.Error("SwapResult.UnmarshalJSON: expected error on malformed JSON")
	}
	var ti TokenInfo
	if err := ti.UnmarshalJSON([]byte("not json")); err == nil {
		t.Error("TokenInfo.UnmarshalJSON: expected error on malformed JSON")
	}
}
