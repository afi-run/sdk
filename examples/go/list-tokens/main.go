// Example 1: List available tokens
//
// GetTokens() accepts an optional network (default: NetworkBase).
// No private key needed — read-only operation.
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
		RPCURL: "https://rpc.ankr.com/base/YOUR_API_KEY",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()

	// List tokens on Base (default)
	baseTokens, err := client.GetTokens(ctx)
	if err != nil {
		log.Fatal("get base tokens:", err)
	}
	fmt.Printf("%d tokens on Base:\n\n", len(baseTokens))
	for _, t := range baseTokens {
		fmt.Printf("  %-10s %s  (%d decimals)\n", t.Symbol, t.Address.Hex(), t.Decimals)
	}

	// List tokens on BSC
	bscTokens, err := client.GetTokens(ctx, afi.NetworkBSC)
	if err != nil {
		log.Fatal("get bsc tokens:", err)
	}
	fmt.Printf("\n%d tokens on BSC:\n\n", len(bscTokens))
	for _, t := range bscTokens {
		fmt.Printf("  %-10s %s  (%d decimals)\n", t.Symbol, t.Address.Hex(), t.Decimals)
	}

	// Find specific tokens by symbol
	var usdc, weth *afi.Token
	for i := range baseTokens {
		switch baseTokens[i].Symbol {
		case "USDC":
			usdc = &baseTokens[i]
		case "WETH":
			weth = &baseTokens[i]
		}
	}
	if usdc == nil || weth == nil {
		log.Fatal("USDC or WETH not found")
	}

	fmt.Printf("\nReady to swap:\n")
	fmt.Printf("  USDC → %s\n", usdc.Address.Hex())
	fmt.Printf("  WETH → %s\n", weth.Address.Hex())
}
