package afi

import "testing"

func TestApplyGasBuffer(t *testing.T) {
	tests := []struct {
		name   string
		gas    uint64
		buffer uint
		want   uint64
	}{
		{"15% on 200k", 200_000, 15, 230_000},
		{"25% on 1M", 1_000_000, 25, 1_250_000},
		{"0% returns input unchanged", 1_000, 0, 1_000},
		{"100% doubles", 50, 100, 100},
		{"non-trivial rounding", 1234, 15, 1419}, // floor((1234 * 115) / 100) = 1419
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyGasBuffer(tt.gas, tt.buffer)
			if got != tt.want {
				t.Errorf("applyGasBuffer(%d, %d) = %d, want %d", tt.gas, tt.buffer, got, tt.want)
			}
		})
	}
}
