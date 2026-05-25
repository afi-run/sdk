// Example 5: Approve only (for apps that manage the flow step by step)
//
// Some apps need to separate the approve step from the swap step —
// for example, to show two distinct prompts to the user.
//
// Approve() approves exactly the amount from the quote — no more.
// If the allowance is already sufficient, no transaction is sent (returns "").
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
	client, err := afi.NewClient(afi.Config{
		RPCURL:     "https://rpc.ankr.com/base/YOUR_API_KEY",
		PrivateKey: "YOUR_PRIVATE_KEY",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()
	usdc := common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913")

	amountIn, err := afi.ParseUnits("200", 6) // 200 USDC
	if err != nil {
		log.Fatal(err)
	}

	// 1. Get quote first — it tells us exactly how much to approve
	quote, err := client.GetQuote(ctx, afi.SwapParams{
		TokenIn:  usdc,
		TokenOut: afi.WETH,
		AmountIn: amountIn,
		Slippage: 0.5,
	})
	if err != nil {
		log.Fatal("get quote:", err)
	}

	// 2. Approve step (wallet prompt #1)
	fmt.Printf("Approving %s USDC...\n", afi.FormatUnits(quote.AmountInWei, 6))
	approveTx, err := client.Approve(ctx, usdc, quote.AmountInWei)
	if err != nil {
		log.Fatal("approve:", err)
	}

	if approveTx == "" {
		fmt.Println("Allowance already sufficient — approval skipped.")
	} else {
		fmt.Printf("Approval tx: %s\n", approveTx)
	}

	// 3. Swap step (wallet prompt #2)
	// ExecuteSwap() re-checks allowance internally before sending,
	// so it's safe to call even if some time passed since Approve.
	fmt.Println("Executing swap...")
	result, err := client.ExecuteSwap(ctx, quote)
	if err != nil {
		log.Fatal("execute swap:", err)
	}

	fmt.Printf("Swap confirmed: %s\n", result.TxHash.Hex())
}
