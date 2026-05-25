// Example 2: Get a quote and inspect it before deciding to swap
//
// GetQuote() only reads data — it does NOT send any transaction.
// Use it to show pricing before the user confirms.
//
// Run: go run ./examples/go/get-quote
package main

import (
	"context"
	"fmt"
	"log"
	"strings"

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

	fmt.Println("Fetching quote for 1000 USDC → WETH...\n")

	quote, err := client.GetQuote(ctx, afi.SwapParams{
		TokenIn:  common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"),
		TokenOut: afi.WETH,
		AmountIn: amountIn,
		Slippage: 0.5,
	})
	if err != nil {
		log.Fatal("get quote:", err)
	}

	path := make([]string, len(quote.Path))
	for i, addr := range quote.Path {
		path[i] = addr.Hex()
	}

	fmt.Println("Quote details:")
	fmt.Printf("  You send:       1000 USDC\n")
	fmt.Printf("  You receive:    ~%s WETH\n", afi.FormatUnits(quote.AmountOutWei, 18))
	fmt.Printf("  Minimum out:    %s WETH  (%.1f%% slippage)\n", afi.FormatUnits(quote.MinOutWei, 18), quote.Slippage)
	fmt.Printf("  Protocol fee:   %.2f%%  (%d bps, read live from contract)\n", float64(quote.FeeBps)/100, quote.FeeBps)
	fmt.Printf("  Route:          %s\n", strings.Join(path, " → "))
	fmt.Printf("  Hops:           %d\n", len(quote.Path)-1)
	fmt.Println("\nQuote is ready. Pass it to ExecuteSwap() to proceed.")
}
