// Example 12: AfiReferralRouter — referral fee + delegation
//
// The AfiReferralRouter wraps Afi.swap and can charge a referral fee of up to
// 0.10% on the OUTPUT token, credited to a referrer and claimed later. It also
// supports delegated swaps (B swaps A's funds; output goes to A).
//
// This example:
//  1. Resolves the router address for the chain.
//  2. Prints swapWithReferral calldata (needs Afi route params + an ERC20
//     approval to the router — see the get-quote / execute-swap examples for
//     how to build `params`).
//  3. Sends a real setDelegateAllowance tx (touches only the caller's own
//     mapping — cheap and safe), then revokes it.
//  4. Prints owner-only setMaxReferralBps calldata (NOT sent).
//
// Prerequisites:
//   - RPC_URL       — Base RPC endpoint
//   - PRIVATE_KEY   — any funded key (the delegation txs spend only gas)
//
// Run: go run ./examples/go/referral-swap
package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"

	afi "github.com/afi-run/sdk/go"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	rpcURL := envOr("RPC_URL", "https://mainnet.base.org")
	pk := mustEnv("PRIVATE_KEY")

	client, err := afi.NewClient(afi.Config{RPCURL: rpcURL, PrivateKey: pk})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	ctx := context.Background()

	// 1. Resolve the router for this chain (Base = 8453).
	router, net, err := afi.ReferralRouterAddress(8453)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Network: %s\nRouter:  %s\nCaller:  %s\n\n", net, router.Hex(), client.Address().Hex())

	usdc := common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913")
	weth := common.HexToAddress("0x4200000000000000000000000000000000000006")
	referrer := common.HexToAddress("0x1111111111111111111111111111111111111111")
	delegate := common.HexToAddress("0x2222222222222222222222222222222222222222")

	// 2. swapWithReferral calldata (not sent — needs real Afi `params` + an
	//    ERC20 approval of `tokenIn` to the router).
	swapData, err := afi.EncodeSwapWithReferral(
		usdc, big.NewInt(1_000_000), weth, big.NewInt(0),
		[]byte{}, referrer, afi.ReferralHardCapBps, // referralBps <= 10
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("swapWithReferral calldata (send to router with a real `params`):\n  0x%x\n\n", swapData)

	// 3. Authorize a delegate to spend up to 5 USDC until a far-future deadline,
	//    then revoke. These only touch the caller's own allowance mapping.
	deadline := uint64(1893456000) // 2030-01-01
	setData, err := afi.EncodeSetDelegateAllowance(usdc, delegate, big.NewInt(5_000_000), deadline)
	if err != nil {
		log.Fatal(err)
	}
	h1, err := sendRawTx(ctx, rpcURL, pk, router, setData)
	if err != nil {
		log.Fatal("setDelegateAllowance:", err)
	}
	fmt.Printf("setDelegateAllowance → %s\n", h1)
	if _, err := client.WaitForTx(ctx, h1); err != nil {
		log.Fatal(err)
	}

	revokeData, err := afi.EncodeRevokeDelegate(usdc, delegate)
	if err != nil {
		log.Fatal(err)
	}
	h2, err := sendRawTx(ctx, rpcURL, pk, router, revokeData)
	if err != nil {
		log.Fatal("revokeDelegate:", err)
	}
	fmt.Printf("revokeDelegate       → %s\n\n", h2)
	if _, err := client.WaitForTx(ctx, h2); err != nil {
		log.Fatal(err)
	}

	// 4. Owner-only: lower the effective referral cap to 5 bps (calldata only).
	capData, err := afi.EncodeReferralSetMaxReferralBps(5)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("setMaxReferralBps(5) calldata (owner-only, NOT sent):\n  0x%x\n", capData)
}

func sendRawTx(ctx context.Context, rpcURL, keyHex string, to common.Address, data []byte) (string, error) {
	eth, err := ethclient.Dial(rpcURL)
	if err != nil {
		return "", err
	}
	defer eth.Close()
	key, err := crypto.HexToECDSA(strings.TrimPrefix(keyHex, "0x"))
	if err != nil {
		return "", err
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
		ChainID: chainID, Nonce: nonce, To: &to,
		Gas: gas * 115 / 100, GasTipCap: tip, GasFeeCap: maxFee, Data: data,
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
		log.Fatalf("Set %s env var", k)
	}
	return v
}
