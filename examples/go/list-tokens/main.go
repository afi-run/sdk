// Example 1: List available tokens on Base
//
// Use GetTokens() to discover which tokens are supported before
// building any swap.
//
// Run: go run ./examples/go/list-tokens
package main

import (
	"context"
	"fmt"
	"log"

	afi "github.com/afi-run/sdk/go"
)

func main() {
	client, err := afi.NewClient(afi.Config{
		RPCURL:     "https://rpc.ankr.com/base/YOUR_API_KEY",
		PrivateKey: "YOUR_PRIVATE_KEY",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	tokens, err := client.GetTokens(context.Background())
	if err != nil {
		log.Fatal("get tokens:", err)
	}

	fmt.Printf("%d tokens available on Base:\n\n", len(tokens))
	for _, t := range tokens {
		fmt.Printf("  %-10s %s  (%d decimals)\n", t.Symbol, t.Address.Hex(), t.Decimals)
	}

	// Find specific tokens by symbol
	var usdc, weth *afi.Token
	for i := range tokens {
		switch tokens[i].Symbol {
		case "USDC":
			usdc = &tokens[i]
		case "WETH":
			weth = &tokens[i]
		}
	}
	if usdc == nil || weth == nil {
		log.Fatal("USDC or WETH not found")
	}

	fmt.Printf("\nReady to swap:\n")
	fmt.Printf("  USDC → %s\n", usdc.Address.Hex())
	fmt.Printf("  WETH → %s\n", weth.Address.Hex())
}
