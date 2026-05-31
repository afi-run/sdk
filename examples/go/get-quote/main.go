// Example 2: Fetch a quote and inspect it before swapping
//
// GetQuote() uses functional options — chain From(), WithSlippage(), etc.
// No private key needed — read-only operation.
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
		RPCURL: "https://rpc.ankr.com/base/YOUR_API_KEY",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()
	usdc := common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913")

	fmt.Println("Fetching quote for 1000 USDC → WETH...\n")

	// Basic quote
	quote, err := client.GetQuote(ctx,
		afi.From(usdc, afi.WETH, "1000"),
		afi.WithSlippage(0.5),
	)
	if err != nil {
		log.Fatal("get quote:", err)
	}

	path := make([]string, len(quote.Path))
	for i, addr := range quote.Path {
		path[i] = addr.Hex()
	}

	fmt.Println("Basic quote:")
	fmt.Printf("  You send:     %s USDC  ($%s)\n", quote.AmountIn, quote.TokenInPrice)
	fmt.Printf("  You receive:  ~%s WETH  ($%s)\n", quote.AmountOut, quote.TokenOutPrice)
	fmt.Printf("  Minimum out:  %s WETH  (%.1f%% slippage)\n", quote.MinOut, quote.Slippage)
	fmt.Printf("  Protocol fee: %.2f%%  (%d bps)\n", float64(quote.FeeBps)/100, quote.FeeBps)
	fmt.Printf("  Route:        %s\n", strings.Join(path, " → "))
	fmt.Printf("  Hops:         %d\n", len(quote.Hops))
	for _, h := range quote.Hops {
		fmt.Printf("    [%s/%s] %s → %s\n", h.Type, h.Kind, h.AmountIn, h.AmountOut)
	}

	// Quote with optional parameters
	fmt.Println("\nQuote with priceBase and specific DEXes:")
	detailedQuote, err := client.GetQuote(ctx,
		afi.From(usdc, afi.WETH, "500"),
		afi.WithSlippage(0.5),
		afi.WithMaxHops(3),
		afi.OnNetwork(afi.NetworkBase),
		afi.WithPriceBase("USDC"),                    // adds TokenInBasePrice / TokenOutBasePrice
		afi.WithDexs(afi.DexUniV3, afi.DexAerodrome), // restrict to these DEXes
	)
	if err != nil {
		log.Fatal("get detailed quote:", err)
	}
	fmt.Printf("  You send:    %s USDC\n", detailedQuote.AmountIn)
	fmt.Printf("  You receive: ~%s WETH\n", detailedQuote.AmountOut)
	if detailedQuote.TokenInBasePrice != "" {
		fmt.Printf("  USDC base price: $%s\n", detailedQuote.TokenInBasePrice)
		fmt.Printf("  WETH base price: $%s\n", detailedQuote.TokenOutBasePrice)
	}

	// Pass a Token object directly (from GetTokens)
	fmt.Println("\nQuote using Token objects from GetTokens():")
	tokens, err := client.GetTokens(ctx)
	if err != nil {
		log.Fatal("get tokens:", err)
	}
	var usdcToken, wethToken *afi.Token
	for i := range tokens {
		switch tokens[i].Symbol {
		case "USDC":
			usdcToken = &tokens[i]
		case "WETH":
			wethToken = &tokens[i]
		}
	}
	if usdcToken == nil || wethToken == nil {
		log.Fatal("tokens not found")
	}
	tokenQuote, err := client.GetQuote(ctx,
		afi.From(*usdcToken, *wethToken, "100"), // pass Token directly
		afi.WithSlippage(1.0),
	)
	if err != nil {
		log.Fatal("token quote:", err)
	}
	fmt.Printf("  %s USDC → ~%s WETH\n", tokenQuote.AmountIn, tokenQuote.AmountOut)
	fmt.Println("\nQuote ready. Pass to ExecuteSwap() to execute.")
}
