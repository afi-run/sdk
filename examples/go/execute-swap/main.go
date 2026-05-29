// Example 3: Quote → review → execute (recommended flow)
//
//  1. Fetch a quote with GetQuote() — no private key needed
//  2. Show pricing to the user
//  3. User confirms — connect the signer
//  4. Call ExecuteSwap(quote) — handles approve + simulate + swap
//
// Run: go run ./examples/go/execute-swap
package main

import (
	"context"
	"errors"
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

	ctx := context.Background()

	// Step 1: Fetch quote — no signer needed
	fmt.Println("Step 1: Fetching quote...")
	quote, err := client.GetQuote(ctx,
		afi.From(
			common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"),
			afi.WETH,
			"1000",
		),
		afi.WithSlippage(0.5),
	)
	if err != nil {
		log.Fatal("get quote:", err)
	}

	// Step 2: Show pricing
	fmt.Printf("\nStep 2: Quote received:\n")
	fmt.Printf("  You send:    %s USDC\n", quote.AmountIn)
	fmt.Printf("  Expected:    ~%s WETH\n", quote.AmountOut)
	fmt.Printf("  Minimum:     %s WETH (%.1f%% slippage)\n", quote.MinOut, quote.Slippage)
	fmt.Printf("  Fee:         %d bps\n\n", quote.FeeBps)

	// Step 3: Connect signer when the user is ready
	if err := client.Connect("YOUR_PRIVATE_KEY"); err != nil {
		log.Fatal("connect:", err)
	}

	// Step 4: Execute with the existing quote (no second fetch needed)
	fmt.Println("Step 3: Executing swap...")
	result, err := client.ExecuteSwap(ctx, quote)
	if err != nil {
		var afiErr *afi.AfiError
		if errors.As(err, &afiErr) {
			switch afiErr.Code {
			case "INSUFFICIENT_BALANCE":
				fmt.Println("Not enough USDC balance.")
			case "SIMULATION_FAILED":
				fmt.Printf("Swap would revert: %s\n", afiErr.Message)
			default:
				fmt.Printf("Swap failed [%s]: %s\n", afiErr.Code, afiErr.Message)
			}
			return
		}
		log.Fatal(err)
	}

	fmt.Println("Swap confirmed!")
	fmt.Printf("  Tx hash:    %s\n", result.TxHash.Hex())
	fmt.Printf("  Block:      %d\n", result.BlockNumber)
	fmt.Printf("  Actual out: %s WETH\n", afi.FormatUnits(result.AmountOut, 18))
	fmt.Printf("  Gas used:   %d\n", result.GasUsed)
}
