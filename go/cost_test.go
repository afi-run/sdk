package afi

import "testing"

func TestSwapCostEstimate_Fields(t *testing.T) {
	// Trivial: ensure the struct can be constructed and fields hold expected types.
	est := &SwapCostEstimate{Gas: 100_000, GasWithBuffer: 115_000}
	if est.Gas == 0 || est.GasWithBuffer == 0 {
		t.Error("expected non-zero values")
	}
}
