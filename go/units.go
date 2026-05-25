package afi

import (
	"fmt"
	"math/big"
	"strings"
)

// FormatUnits converts a raw wei amount to a human-readable decimal string.
//
//	FormatUnits(big.NewInt(1000_000000), 6) // "1000"
//	FormatUnits(big.NewInt(1_500000), 6)    // "1.5"
func FormatUnits(amount *big.Int, decimals uint8) string {
	return formatAmount(amount, decimals)
}

// ParseUnits converts a human-readable decimal string to raw wei.
//
//	ParseUnits("1000", 6)   // big.NewInt(1000_000000)
//	ParseUnits("1.5", 6)    // big.NewInt(1_500000)
func ParseUnits(amount string, decimals uint8) (*big.Int, error) {
	parts := strings.SplitN(amount, ".", 2)
	whole := parts[0]
	frac := ""
	if len(parts) == 2 {
		frac = parts[1]
	}

	if len(frac) > int(decimals) {
		return nil, fmt.Errorf("too many decimal places: %q has %d, max %d", amount, len(frac), decimals)
	}

	frac = frac + strings.Repeat("0", int(decimals)-len(frac))

	combined := whole + frac
	result := new(big.Int)
	if _, ok := result.SetString(combined, 10); !ok {
		return nil, fmt.Errorf("invalid amount: %q", amount)
	}
	return result, nil
}
