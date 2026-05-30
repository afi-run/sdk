package afi

import (
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// ZeroAddress is the canonical all-zeros address.
var ZeroAddress = common.Address{}

// ZeroAddressHex is the lowercase string form of ZeroAddress.
const ZeroAddressHex = "0x0000000000000000000000000000000000000000"

// IsAddress reports whether s is a syntactically valid 20-byte hex address.
// Requires the "0x" prefix (unlike go-ethereum's common.IsHexAddress which is lenient)
// to match the behavior of viem/ethers and the TypeScript SDK.
// Does NOT validate the EIP-55 checksum — use Checksum to inspect that.
func IsAddress(s string) bool {
	if len(s) < 2 || (s[0] != '0' || (s[1] != 'x' && s[1] != 'X')) {
		return false
	}
	return common.IsHexAddress(s)
}

// Checksum returns the EIP-55 checksummed form of s. Returns the zero address
// when the input is not a valid hex address.
func Checksum(s string) string {
	if !common.IsHexAddress(s) {
		return ZeroAddressHex
	}
	return common.HexToAddress(s).Hex()
}

// IsZeroAddress reports whether s equals the zero address (case-insensitive).
func IsZeroAddress(s string) bool {
	return strings.EqualFold(s, ZeroAddressHex)
}

// EqualAddresses compares two addresses case-insensitively. Returns false for
// strings of different lengths.
func EqualAddresses(a, b string) bool {
	return strings.EqualFold(a, b)
}
