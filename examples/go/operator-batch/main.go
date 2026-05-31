// Example 6: Operator batchSwapFor — execute swaps on behalf of pre-approved users.
//
// The Afi router exposes `swapFor(user, ...)` and `batchSwapFor(tuple[])` which
// can only be called by an account registered as an operator on the contract.
// Each user must have approved the Afi router for `amountIn` of `tokenIn`
// ahead of time.
//
// Prerequisites:
//   - RPC_URL                — Base RPC endpoint
//   - OPERATOR_PRIVATE_KEY   — one of the 7 registered operators on Afi (Base)
//   - USER1_ADDR, USER2_ADDR, USER3_ADDR — pre-approved users
//
// Expected output: one tx hash, plus a per-user SwapExecuted row parsed from
// the receipt logs.
//
// NOTE: high-level `client.BatchSwapFor()` is not yet exposed by the Go SDK —
// this example signs and sends a raw tx via go-ethereum's ethclient.
// TODO: client.BatchSwapFor when SDK lands it.
//
// Run: go run ./examples/go/operator-batch
package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"

	afi "github.com/afi-run/sdk/go"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Mirror the on-chain SwapRequest tuple from AFI_ABI.
type SwapRequest struct {
	User     common.Address `abi:"user"`
	TokenIn  common.Address `abi:"tokenIn"`
	AmountIn *big.Int       `abi:"amountIn"`
	TokenOut common.Address `abi:"tokenOut"`
	MinOut   *big.Int       `abi:"minOut"`
	Params   []byte         `abi:"params"`
}

func main() {
	rpcURL := envOr("RPC_URL", "https://mainnet.base.org")
	opKey := mustEnv("OPERATOR_PRIVATE_KEY")
	user1 := common.HexToAddress(envOr("USER1_ADDR", "0x0000000000000000000000000000000000000001"))
	user2 := common.HexToAddress(envOr("USER2_ADDR", "0x0000000000000000000000000000000000000002"))
	user3 := common.HexToAddress(envOr("USER3_ADDR", "0x0000000000000000000000000000000000000003"))

	client, err := afi.NewClient(afi.Config{RPCURL: rpcURL, PrivateKey: opKey})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()
	usdc := common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913")
	afiAddr := afi.AfiAddresses[afi.NetworkBase]

	fmt.Printf("Operator: %s\nAFI:      %s\n\n", client.Address().Hex(), afiAddr.Hex())

	// Build one SwapRequest per user from a SDK quote.
	users := []common.Address{user1, user2, user3}
	var reqs []SwapRequest
	for _, u := range users {
		q, err := client.GetQuote(ctx,
			afi.From(usdc, afi.WETH, "100"),
			afi.WithSlippage(0.5),
		)
		if err != nil {
			log.Fatal("quote:", err)
		}
		fmt.Printf("Quote for %s: %s USDC → ~%s WETH\n", u.Hex(), q.AmountIn, q.AmountOut)
		reqs = append(reqs, SwapRequest{
			User: u, TokenIn: usdc, AmountIn: q.AmountInWei,
			TokenOut: afi.WETH, MinOut: q.MinOutWei, Params: q.Steps,
		})
	}

	// Pack batchSwapFor calldata via the AFI ABI.
	parsedABI, err := abi.JSON(strings.NewReader(afi.AfiABI))
	if err != nil {
		log.Fatal("parse abi:", err)
	}
	data, err := parsedABI.Pack("batchSwapFor", reqs)
	if err != nil {
		log.Fatal("pack:", err)
	}

	txHash, err := sendRawTx(ctx, rpcURL, opKey, afiAddr, data)
	if err != nil {
		log.Fatal("send:", err)
	}
	fmt.Printf("\nbatchSwapFor submitted: %s\n", txHash)

	rcpt, err := client.WaitForTx(ctx, txHash)
	if err != nil {
		log.Fatal("wait:", err)
	}
	fmt.Printf("Confirmed in block %d, gas used %d\n", rcpt.BlockNumber, rcpt.GasUsed)

	// Parse the receipt logs into SwapExecuted events.
	eth, err := ethclient.Dial(rpcURL)
	if err != nil {
		log.Fatal(err)
	}
	defer eth.Close()
	rec, err := eth.TransactionReceipt(ctx, common.HexToHash(txHash))
	if err != nil {
		log.Fatal("receipt:", err)
	}
	events, err := afi.ParseSwapExecuted(rec.Logs)
	if err != nil {
		log.Fatal("parse:", err)
	}
	fmt.Printf("\n%d SwapExecuted events:\n", len(events))
	for _, ev := range events {
		fmt.Printf("  %s  %s USDC → %s WETH\n",
			ev.From.Hex(),
			afi.FormatUnits(ev.AmountIn, 6),
			afi.FormatUnits(ev.AmountOut, 18),
		)
	}
}

// sendRawTx signs and broadcasts a raw EIP-1559 tx — the SDK doesn't expose a
// public send-arbitrary-calldata method yet. Replace with the SDK helper when
// it lands.
func sendRawTx(ctx context.Context, rpcURL, keyHex string, to common.Address, data []byte) (string, error) {
	eth, err := ethclient.Dial(rpcURL)
	if err != nil {
		return "", err
	}
	defer eth.Close()

	key, err := crypto.HexToECDSA(strings.TrimPrefix(keyHex, "0x"))
	if err != nil {
		return "", fmt.Errorf("key: %w", err)
	}
	from := crypto.PubkeyToAddress(key.PublicKey)

	chainID, err := eth.ChainID(ctx)
	if err != nil {
		return "", err
	}
	nonce, err := eth.PendingNonceAt(ctx, from)
	if err != nil {
		return "", err
	}
	gas, err := eth.EstimateGas(ctx, ethereum.CallMsg{From: from, To: &to, Data: data})
	if err != nil {
		return "", fmt.Errorf("estimate gas: %w", err)
	}
	tip, err := eth.SuggestGasTipCap(ctx)
	if err != nil {
		return "", err
	}
	head, err := eth.HeaderByNumber(ctx, nil)
	if err != nil {
		return "", err
	}
	maxFee := new(big.Int).Add(new(big.Int).Mul(head.BaseFee, big.NewInt(2)), tip)

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		To:        &to,
		Gas:       gas * 115 / 100,
		GasTipCap: tip,
		GasFeeCap: maxFee,
		Data:      data,
	})
	signed, err := types.SignTx(tx, types.NewLondonSigner(chainID), key)
	if err != nil {
		return "", err
	}
	if err := eth.SendTransaction(ctx, signed); err != nil {
		return "", err
	}
	return signed.Hash().Hex(), nil
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

// silence unused import in some builds.
var _ = ecdsa.PrivateKey{}
