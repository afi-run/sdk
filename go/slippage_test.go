package afi

import (
	"math/big"
	"testing"
)

func TestApplySlippage(t *testing.T) {
	cases := []struct {
		name     string
		amount   *big.Int
		slippage float64
		want     *big.Int
	}{
		{"0.5%", big.NewInt(10_000), 0.5, big.NewInt(9_950)},
		{"1.0%", big.NewInt(10_000), 1.0, big.NewInt(9_900)},
		{"5.0%", big.NewInt(10_000), 5.0, big.NewInt(9_500)},
		{"1.25%", big.NewInt(10_000), 1.25, big.NewInt(9_875)},
		{"0.01%", big.NewInt(10_000), 0.01, big.NewInt(9_999)},
		{"0% returns unchanged", big.NewInt(12345), 0, big.NewInt(12345)},
		{"negative clamps to 0", big.NewInt(10_000), -1.0, big.NewInt(10_000)},
		{"100% returns 0", big.NewInt(10_000), 100, big.NewInt(0)},
		{"150% returns 0", big.NewInt(10_000), 150, big.NewInt(0)},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ApplySlippage(c.amount, c.slippage)
			if got.Cmp(c.want) != 0 {
				t.Errorf("ApplySlippage(%s, %v) = %s, want %s", c.amount, c.slippage, got, c.want)
			}
		})
	}
}

func TestApplySlippage_LargeWei(t *testing.T) {
	oneEth, _ := new(big.Int).SetString("1000000000000000000", 10)
	got := ApplySlippage(oneEth, 0.5)
	want, _ := new(big.Int).SetString("995000000000000000", 10)
	if got.Cmp(want) != 0 {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestCalculateMinOut_AliasOfApplySlippage(t *testing.T) {
	amount := big.NewInt(10_000)
	a := CalculateMinOut(amount, 0.5)
	b := ApplySlippage(amount, 0.5)
	if a.Cmp(b) != 0 {
		t.Error("CalculateMinOut should equal ApplySlippage")
	}
}
