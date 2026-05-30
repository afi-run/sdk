package afi

import (
	"math"
	"math/big"
)

// ApplySlippage returns amount * (1 - slippagePct/100), floor-divided.
// `slippagePct` is in percent units (0.5 = 0.5%, 1.25 = 1.25%). Negative values
// are clamped to 0 (no slippage). Values >= 100 return zero.
func ApplySlippage(amount *big.Int, slippagePct float64) *big.Int {
	bps := int64(math.Round(slippagePct * 100))
	if bps <= 0 {
		return new(big.Int).Set(amount)
	}
	if bps >= 10_000 {
		return new(big.Int)
	}
	mul := new(big.Int).Mul(amount, big.NewInt(10_000-bps))
	return new(big.Int).Quo(mul, big.NewInt(10_000))
}

// CalculateMinOut is an alias of ApplySlippage that reads as `minOut`
// when you have a raw `amountOutWei`.
func CalculateMinOut(amountOutWei *big.Int, slippagePct float64) *big.Int {
	return ApplySlippage(amountOutWei, slippagePct)
}
