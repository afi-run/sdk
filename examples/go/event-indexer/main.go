// Example 10: On-chain event indexer
//
// Reads the last N blocks from Base and tabulates the protocol's main events:
//   - SwapExecuted        (Afi)
//   - FeeCollected        (Afi)
//   - FlashLoanExecuted   (NMR)
//   - ProfitSwept         (NMR)
//
// Uses go-ethereum FilterLogs per contract address, then runs each batch
// through the SDK's typed parsers.
//
// Prerequisites:
//   - RPC_URL                  — Base RPC endpoint
//   - BLOCK_RANGE (optional)   — how many blocks back from `latest` to scan (default: 5000)
//
// Expected output: aggregate counts and a tail of recent rows per event type.
//
// Run: go run ./examples/go/event-indexer
package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"

	afi "github.com/afi-run/sdk/go"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	rpcURL := envOr("RPC_URL", "https://mainnet.base.org")
	blockRange := int64(5000)
	if v := os.Getenv("BLOCK_RANGE"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			blockRange = n
		}
	}

	eth, err := ethclient.Dial(rpcURL)
	if err != nil {
		log.Fatal(err)
	}
	defer eth.Close()

	ctx := context.Background()
	tip, err := eth.BlockNumber(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fromBlock := int64(tip) - blockRange + 1
	if fromBlock < 0 {
		fromBlock = 0
	}
	fmt.Printf("Scanning blocks %d → %d (%d blocks)\n\n", fromBlock, tip, blockRange)

	afiAddr := afi.AfiAddresses[afi.NetworkBase]
	nmrAddr := afi.NMRAddresses[afi.NetworkBase]

	afiLogs, err := getLogs(ctx, eth, afiAddr, fromBlock, int64(tip))
	if err != nil {
		log.Fatal("afi logs:", err)
	}
	nmrLogs, err := getLogs(ctx, eth, nmrAddr, fromBlock, int64(tip))
	if err != nil {
		log.Fatal("nmr logs:", err)
	}

	swaps, _ := afi.ParseSwapExecuted(afiLogs)
	feeCollects, _ := afi.ParseFeeCollected(afiLogs)
	flashLoans, _ := afi.ParseFlashLoanExecuted(nmrLogs)
	profitSweeps, _ := afi.ParseProfitSwept(nmrLogs)

	fmt.Printf("Event counts:\n")
	fmt.Printf("  SwapExecuted        %d\n", len(swaps))
	fmt.Printf("  FeeCollected        %d\n", len(feeCollects))
	fmt.Printf("  FlashLoanExecuted   %d\n", len(flashLoans))
	fmt.Printf("  ProfitSwept         %d\n\n", len(profitSweeps))

	if len(swaps) > 0 {
		fmt.Println("Recent SwapExecuted:")
		for _, s := range tail(swaps, 3) {
			fmt.Printf("  %s %s → %s amount=%s/%s\n",
				s.From.Hex(), s.AssetIn.Hex(), s.AssetOut.Hex(),
				s.AmountIn.String(), s.AmountOut.String())
		}
	}
	if len(flashLoans) > 0 {
		fmt.Println("\nRecent FlashLoanExecuted:")
		for _, f := range tail(flashLoans, 3) {
			fmt.Printf("  %s profit=%s premium=%s\n",
				f.Asset.Hex(),
				afi.FormatUnits(f.Profit, 6),
				afi.FormatUnits(f.Premium, 6))
		}
	}
	if len(profitSweeps) > 0 {
		fmt.Println("\nRecent ProfitSwept:")
		for _, p := range tail(profitSweeps, 3) {
			fmt.Printf("  %s → %s amount=%s\n",
				p.Asset.Hex(), p.To.Hex(), p.Amount.String())
		}
	}
}

func getLogs(ctx context.Context, eth *ethclient.Client, addr common.Address, from, to int64) ([]*types.Log, error) {
	raw, err := eth.FilterLogs(ctx, ethereum.FilterQuery{
		Addresses: []common.Address{addr},
		FromBlock: big.NewInt(from),
		ToBlock:   big.NewInt(to),
	})
	if err != nil {
		return nil, err
	}
	out := make([]*types.Log, len(raw))
	for i := range raw {
		out[i] = &raw[i]
	}
	return out, nil
}

func tail[T any](xs []T, n int) []T {
	if len(xs) <= n {
		return xs
	}
	return xs[len(xs)-n:]
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
