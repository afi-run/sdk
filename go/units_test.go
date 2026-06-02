package afi

import (
	"math/big"
	"testing"
)

func TestFormatUnits(t *testing.T) {
	tests := []struct {
		name     string
		amount   *big.Int
		decimals uint8
		want     string
	}{
		{"whole USDC", big.NewInt(1000_000000), 6, "1000"},
		{"fractional 1.5", big.NewInt(1_500000), 6, "1.5"},
		{"sub-unit", big.NewInt(123456), 6, "0.123456"},
		{"whole WETH 18 dec", new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil), 18, "1"},
		{"half WETH", new(big.Int).Div(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil), big.NewInt(2)), 18, "0.5"},
		{"zero", big.NewInt(0), 6, "0"},
		{"negative sub-unit", big.NewInt(-362), 6, "-0.000362"},
		{"negative fractional", big.NewInt(-1_500000), 6, "-1.5"},
		{"negative whole", big.NewInt(-1000_000000), 6, "-1000"},
		{"negative sub-unit no trailing zeros", big.NewInt(-123456), 6, "-0.123456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatUnits(tt.amount, tt.decimals)
			if got != tt.want {
				t.Errorf("FormatUnits(%s, %d) = %q, want %q", tt.amount, tt.decimals, got, tt.want)
			}
		})
	}
}

func TestParseUnits(t *testing.T) {
	tests := []struct {
		name     string
		amount   string
		decimals uint8
		want     *big.Int
		wantErr  bool
	}{
		{"whole USDC", "1000", 6, big.NewInt(1000_000000), false},
		{"fractional 1.5", "1.5", 6, big.NewInt(1_500000), false},
		{"sub-unit 0.123456", "0.123456", 6, big.NewInt(123456), false},
		{"whole WETH", "1", 18, new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil), false},
		{"half WETH", "0.5", 18, new(big.Int).Div(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil), big.NewInt(2)), false},
		{"pads missing decimals", "1.1", 6, big.NewInt(1_100000), false},
		{"zero", "0", 6, big.NewInt(0), false},
		{"too many decimal places", "1.1234567", 6, nil, true},
		{"invalid input", "abc", 6, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseUnits(tt.amount, tt.decimals)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Cmp(tt.want) != 0 {
				t.Errorf("ParseUnits(%q, %d) = %s, want %s", tt.amount, tt.decimals, got, tt.want)
			}
		})
	}
}

func TestParseUnitsRoundtrip(t *testing.T) {
	raw := big.NewInt(1_234500)
	formatted := FormatUnits(raw, 6)
	parsed, err := ParseUnits(formatted, 6)
	if err != nil {
		t.Fatalf("roundtrip ParseUnits error: %v", err)
	}
	if parsed.Cmp(raw) != 0 {
		t.Errorf("roundtrip: got %s, want %s", parsed, raw)
	}
}
