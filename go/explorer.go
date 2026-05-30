package afi

import (
	"fmt"
	"strings"
)

func explorerBase(network Network, override ...string) (string, error) {
	if len(override) > 0 && override[0] != "" {
		return strings.TrimRight(override[0], "/"), nil
	}
	base, ok := NetworkExplorers[network]
	if !ok {
		return "", fmt.Errorf("no explorer URL configured for network %q", network)
	}
	return strings.TrimRight(base, "/"), nil
}

// TxURL returns the explorer URL for a transaction hash on `network`.
// Pass an explicit `explorer` base to override the default in NetworkExplorers.
//
//	afi.TxURL("0xabc...", afi.NetworkBase)                 // https://basescan.org/tx/0xabc...
//	afi.TxURL(hash, afi.NetworkBase, "https://customexp")  // custom base
func TxURL(hash string, network Network, explorer ...string) (string, error) {
	base, err := explorerBase(network, explorer...)
	if err != nil {
		return "", err
	}
	return base + "/tx/" + hash, nil
}

// AddressURL returns the explorer URL for an address on `network`.
func AddressURL(address string, network Network, explorer ...string) (string, error) {
	base, err := explorerBase(network, explorer...)
	if err != nil {
		return "", err
	}
	return base + "/address/" + address, nil
}
