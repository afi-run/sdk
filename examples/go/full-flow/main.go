// Example 4: One-call convenience flow using Swap()
//
// Swap() is equivalent to GetQuote() + ExecuteSwap() in sequence.
// Use for bots and scripts where you don't need to review the quote.
//
// For user-facing apps, prefer example 3 (quote → review → execute).
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
		RPCURL: "https://rpc.ankr.com/base/YOUR_API_KEY",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	if err := client.Connect("YOUR_PRIVATE_KEY"); err != nil {
		log.Fatal("connect:", err)
	}

	ctx := context.Background()

	// Variation A: minimal
	result, err := client.Swap(ctx,
		afi.From(
			common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"),
			afi.WETH,
			"500",
		),
		afi.WithSlippage(1.0),
	)
	if err != nil {
		log.Fatal("swap:", err)
	}

	fmt.Println("Swap completed!")
	fmt.Printf("  Tx:         %s\n", result.TxHash.Hex())
	fmt.Printf("  Amount in:  %s USDC\n", afi.FormatUnits(result.AmountIn, 6))
	fmt.Printf("  Amount out: %s WETH\n", afi.FormatUnits(result.AmountOut, 18))

	// Variation B: with optional parameters
	// result2, err := client.Swap(ctx,
	// 	afi.From(usdc, afi.WETH, "200"),
	// 	afi.WithSlippage(0.5),
	// 	afi.WithMaxHops(3),
	// 	afi.OnNetwork(afi.NetworkBase),
	// 	afi.WithDexs(afi.DexUniV3, afi.DexAerodrome),
	// )
}
