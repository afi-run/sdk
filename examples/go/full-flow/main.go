// Example 4: Full flow using Swap() convenience method
//
// Swap() is a shorthand that calls GetQuote() + ExecuteSwap() in sequence.
// Use it when you don't need to inspect the quote before executing,
// e.g. in automated bots or scripts.
//
// Run: go run ./examples/go/full-flow
package main

import (
	"context"
	"fmt"
	"log"

	afi "github.com/afi-run/sdk/go"
	"github.com/ethereum/go-ethereum/common"
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

	amountIn, err := afi.ParseUnits("500", 6) // 500 USDC
	if err != nil {
		log.Fatal(err)
	}

	result, err := client.Swap(context.Background(), afi.SwapParams{
		TokenIn:  common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"),
		TokenOut: afi.WETH,
		AmountIn: amountIn,
		Slippage: 1.0, // 1%
	})
	if err != nil {
		log.Fatal("swap:", err)
	}

	fmt.Println("Swap completed!")
	fmt.Printf("  Tx:         %s\n", result.TxHash.Hex())
	fmt.Printf("  Amount in:  %s USDC\n", afi.FormatUnits(result.AmountIn, 6))
	fmt.Printf("  Amount out: %s WETH\n", afi.FormatUnits(result.AmountOut, 18))
}
