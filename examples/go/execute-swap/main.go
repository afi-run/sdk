// Example 3: Quote → review → execute (recommended flow)
//
// This is the recommended pattern for production applications:
//  1. Fetch a quote (no tx sent)
//  2. Show pricing to the user
//  3. User confirms
//  4. Call ExecuteSwap(quote) — handles approve + simulate + swap
//
// ExecuteSwap() will:
//   - Check your token balance
//   - Approve exactly the input amount to the AFI contract (skipped if already approved)
//   - Simulate the swap via eth_call — returns error before sending if it would revert
//   - Send the swap transaction
//   - Return confirmed amounts from the on-chain SwapExecuted event
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
		RPCURL:     "https://rpc.ankr.com/base/YOUR_API_KEY",
		PrivateKey: "YOUR_PRIVATE_KEY",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()

	amountIn, err := afi.ParseUnits("1000", 6) // 1000 USDC
	if err != nil {
		log.Fatal(err)
	}

	// Step 1: Get quote (read-only, no tx)
	fmt.Println("Step 1: Fetching quote...")
	quote, err := client.GetQuote(ctx, afi.SwapParams{
		TokenIn:  common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"),
		TokenOut: afi.WETH,
		AmountIn: amountIn,
		Slippage: 0.5,
	})
	if err != nil {
		log.Fatal("get quote:", err)
	}

	// Step 2: Show pricing
	fmt.Printf("\nStep 2: Quote received:\n")
	fmt.Printf("  Expected:  ~%s WETH\n", afi.FormatUnits(quote.AmountOutWei, 18))
	fmt.Printf("  Minimum:   %s WETH (%.1f%% slippage)\n", afi.FormatUnits(quote.MinOutWei, 18), quote.Slippage)
	fmt.Printf("  Fee:       %d bps\n\n", quote.FeeBps)

	// Step 3: Execute with the existing quote (no second fetch needed)
	fmt.Println("Step 3: Executing swap...")
	result, err := client.ExecuteSwap(ctx, quote)
	if err != nil {
		var afiErr *afi.AfiError
		if errors.As(err, &afiErr) {
			switch afiErr.Code {
			case "INSUFFICIENT_BALANCE":
				fmt.Println("Not enough USDC balance.")
			case "SIMULATION_FAILED":
				// No tx was sent, no gas wasted
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
	fmt.Printf("  Actual out: %s WETH  (from on-chain event)\n", afi.FormatUnits(result.AmountOut, 18))
	fmt.Printf("  Gas used:   %d\n", result.GasUsed)
}
