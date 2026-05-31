// Example 7: NMR flash-loan arbitrage (operator-only)
//
// Flow:
//  1. Client.FindArbitrage queries afi-rpc /arbitrage for candidate routes of a
//     cycle (tokenIn == tokenOut == asset).
//  2. Pick the most profitable route and build the Afi.swap params with
//     afi.QuoteFromRoute (wraps the route's encoded hop).
//  3. Client.NMRArbitrage calls NMR.requestOperation(asset, amount, params) —
//     Aave flash-loan + cycle, signed by a registered operator.
//  4. Parse the FlashLoanExecuted event and log realized profit.
//
// Prerequisites:
//   - RPC_URL                — Base RPC endpoint
//   - OPERATOR_PRIVATE_KEY   — one of the registered operators on NMR (Base)
//   - AFI_API_URL            — afi-rpc base URL (default: https://rpc.afi.run)
//
// Run: go run ./examples/go/nmr-arbitrage
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	afi "github.com/afi-run/sdk/go"
	"github.com/ethereum/go-ethereum/common"
)

func main() {
	rpcURL := envOr("RPC_URL", "https://mainnet.base.org")
	apiURL := envOr("AFI_API_URL", "https://rpc.afi.run")
	opKey := mustEnv("OPERATOR_PRIVATE_KEY")

	client, err := afi.NewClient(afi.Config{RPCURL: rpcURL, PrivateKey: opKey})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	client.SetApiURL(apiURL)

	ctx := context.Background()
	usdc := common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913")
	nmrAddr := afi.NMRAddresses[afi.NetworkBase]

	fmt.Printf("Operator: %s\nNMR:      %s\n\n", client.Address().Hex(), nmrAddr.Hex())

	// 1. Discover candidate cycle routes (tokenIn == tokenOut == USDC).
	routes, err := client.FindArbitrage(ctx, afi.ArbitrageRequest{
		"network":  string(afi.NetworkBase),
		"tokenIn":  usdc.Hex(),
		"tokenOut": usdc.Hex(),
		"amountIn": "1000",
		"slippage": 0.5,
	})
	if err != nil {
		log.Fatal("findArbitrage:", err)
	}

	// 2. Pick the most profitable route.
	var best *afi.RouteQuote
	for i := range routes {
		p := routes[i].Profit()
		if p == nil || p.Sign() <= 0 {
			continue
		}
		if best == nil || p.Cmp(best.Profit()) > 0 {
			best = &routes[i]
		}
	}
	if best == nil {
		fmt.Println("No profitable arbitrage opportunity right now.")
		return
	}
	fmt.Printf("Best route: dex=%s id=%d, est. profit=%s (base units)\n",
		best.Kind, best.RouteID, best.Profit())

	q, err := afi.QuoteFromRoute(*best, nil)
	if err != nil {
		log.Fatal("build quote:", err)
	}

	// 3. Broadcast NMR.requestOperation via the high-level wrapper.
	rcpt, err := client.NMRArbitrage(ctx, q.TokenIn, q.AmountInWei, q.Steps)
	if err != nil {
		log.Fatal("requestOperation:", err)
	}
	fmt.Printf("\nrequestOperation: %s\nConfirmed in block %d, gas %d\n",
		rcpt.TxHash.Hex(), rcpt.BlockNumber, rcpt.GasUsed)

	// 4. Parse the result events.
	if failed, _ := afi.ParseFlashLoanFailed(rcpt.Logs); len(failed) > 0 {
		fmt.Printf("FlashLoanFailed: %s\n", failed[0].Reason)
		return
	}
	for _, ev := range mustParse(afi.ParseFlashLoanExecuted(rcpt.Logs)) {
		fmt.Println("\nFlashLoanExecuted:")
		fmt.Printf("  asset:   %s\n", ev.Asset.Hex())
		fmt.Printf("  amount:  %s USDC\n", afi.FormatUnits(ev.Amount, 6))
		fmt.Printf("  premium: %s USDC\n", afi.FormatUnits(ev.Premium, 6))
		fmt.Printf("  profit:  %s USDC\n", afi.FormatUnits(ev.Profit, 6))
	}
}

func mustParse(ev []afi.FlashLoanExecutedEvent, err error) []afi.FlashLoanExecutedEvent {
	if err != nil {
		log.Fatal("parse FlashLoanExecuted:", err)
	}
	return ev
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func mustEnv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("env %s is required", k)
	}
	return v
}
