// Example 5: Staged flow — full control over each step
//
//  1. Fetch quote with GetQuote()
//  2. Connect signer
//  3. Approve  — get txHash immediately, wait separately
//  4. Simulate — dry-run before spending gas
//  5. Submit   — send the swap tx, get txHash immediately
//  6. Wait     — block until confirmed, get SwapResult
//
// Use this in apps that need step-by-step wallet prompts and progress UI.
//
// Run: go run ./examples/go/approve-only
package main

import (
	"context"
	"fmt"
	"log"

	afi "github.com/afi-run/sdk/go"
	"github.com/ethereum/go-ethereum/common"
)

func main() {
	// Step 1: Read-only client — no private key needed for quotes
	client, err := afi.NewClient(afi.Config{
		RPCURL: "https://rpc.ankr.com/base/YOUR_API_KEY",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()
	usdc := common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913")

	// Step 2: Fetch quote — no signer required
	quote, err := client.GetQuote(ctx,
		afi.From(usdc, afi.WETH, "200"),
		afi.WithSlippage(0.5),
		afi.OnNetwork(afi.NetworkBase),
	)
	if err != nil {
		log.Fatal("get quote:", err)
	}

	fmt.Printf("Quote: %s USDC → ~%s WETH\n", quote.AmountIn, quote.AmountOut)
	fmt.Printf("Minimum: %s WETH  (%.1f%% slippage)\n", quote.MinOut, quote.Slippage)

	// Step 3: Connect signer when the user is ready to transact
	if err := client.Connect("YOUR_PRIVATE_KEY"); err != nil {
		log.Fatal("connect:", err)
	}

	// Optional: pull symbol/decimals/balance/allowance for USDC in a single multicall.
	info, err := client.MyTokenInfo(ctx, usdc)
	if err != nil {
		log.Fatal("token info:", err)
	}
	fmt.Printf("%s (%d decimals) — balance: %s, allowance: %s\n",
		info.Symbol, info.Decimals, info.Balance, info.Allowance)

	// Step 4: Approve only if the AFI contract isn't already allowed to spend amountIn.
	already, err := client.HasAllowance(ctx, usdc, quote.AmountInWei)
	if err != nil {
		log.Fatal("has allowance:", err)
	}
	if already {
		fmt.Println("Allowance already sufficient — approval skipped.")
	} else {
		approval, err := client.Approve(ctx, usdc, quote.AmountInWei)
		if err != nil {
			log.Fatal("approve:", err)
		}
		if approval != nil {
			fmt.Printf("Approval submitted: %s\n", approval.TxHash)
			approvalReceipt, err := approval.Wait(ctx)
			if err != nil {
				log.Fatal("wait approval:", err)
			}
			fmt.Printf("Approval confirmed in block %d\n", approvalReceipt.BlockNumber)
		}
	}

	// Step 5: Simulate — returns the revert reason as an *AfiError when the swap would fail.
	if err := client.Simulate(ctx, quote); err != nil {
		fmt.Printf("Swap would revert: %s — cancelled.\n", err)
		return
	}

	// Step 6: Submit — send the swap tx, get txHash immediately
	pending, err := client.SubmitSwap(ctx, quote)
	if err != nil {
		log.Fatal("submit swap:", err)
	}
	fmt.Printf("Swap submitted: %s\n", pending.TxHash)

	// Step 7: Wait for on-chain confirmation
	result, err := pending.Wait(ctx)
	if err != nil {
		log.Fatal("wait swap:", err)
	}

	fmt.Println("\nSwap confirmed!")
	fmt.Printf("  Tx hash:    %s\n", result.TxHash.Hex())
	fmt.Printf("  Block:      %d\n", result.BlockNumber)
	fmt.Printf("  Amount out: %s WETH\n", afi.FormatUnits(result.AmountOut, 18))
	fmt.Printf("  Gas used:   %d\n", result.GasUsed)
}
