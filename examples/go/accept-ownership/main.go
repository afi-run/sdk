// Example 11: Accept ownership of all AFI contracts (Ownable2Step handover)
//
// The Solidity script `script/TransferOwnership.s.sol` only INITIATES the
// Ownable2Step transfer for each contract. Until the new owner calls
// `acceptOwnership()` on each one, current ownership stays with the deployer.
//
// This example assumes the running key (NEW_OWNER_PRIVATE_KEY) is the pending
// owner. It:
//  1. Reads pendingOwner() on the AFI router to confirm we are it.
//  2. Sends acceptOwnership() to all 11 contracts that were transferred:
//     RouteRegistry, Afi, CakeV3, UniV3, UniV4, Aerodrome,
//     BalancerV3, Curve128, Curve256, FluidDex, AaveLiquidator.
//
// For Safe / multi-sig owners, you'd batch these in the Safe UI instead —
// this example targets the EOA flow only.
//
// Prerequisites:
//   - RPC_URL                  — Base RPC endpoint
//   - NEW_OWNER_PRIVATE_KEY    — pending owner key for every contract
//   - ROUTE_REGISTRY, AFI, CAKEV3, UNIV3, UNIV4, AERODROME, BALANCERV3,
//     CURVE128, CURVE256, FLUIDDEX, AAVE_LIQUIDATOR  — addresses to accept on
//     (missing addresses are skipped with a log line)
//
// Expected output: pending-owner confirmation, then one tx hash per contract.
//
// NOTE: convenience wrappers `client.AcceptOwnership(addr)` /
// `AcceptOwnershipBatch(addrs)` are not exposed yet — this example builds the
// calldata locally from a minimal Ownable2Step ABI and broadcasts a raw tx.
// TODO: client.AcceptOwnership / client.AcceptOwnershipBatch when SDK lands them.
//
// Run: go run ./examples/go/accept-ownership
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
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Minimal Ownable2Step ABI — mirrors OWNABLE2STEP_ABI in the Node SDK.
const ownable2StepABI = `[
  {"type":"function","name":"acceptOwnership","stateMutability":"nonpayable","inputs":[],"outputs":[]},
  {"type":"function","name":"pendingOwner","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]}
]`

type target struct {
	name string
	addr common.Address
}

func main() {
	rpcURL := envOr("RPC_URL", "https://mainnet.base.org")
	ownerKey := mustEnv("NEW_OWNER_PRIVATE_KEY")

	client, err := afi.NewClient(afi.Config{RPCURL: rpcURL, PrivateKey: ownerKey})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()
	newOwner := client.Address()
	fmt.Printf("New owner: %s\n", newOwner.Hex())

	// Step 1: sanity check — read pendingOwner() on the AFI router.
	afiAddr := afi.AfiAddresses[afi.NetworkBase]
	pending, err := client.GetPendingOwner(ctx, afiAddr, afi.BaseChainID)
	if err != nil {
		log.Fatal("getPendingOwner:", err)
	}
	fmt.Printf("AFI router pendingOwner(): %s\n", pending.Hex())
	if !strings.EqualFold(pending.Hex(), newOwner.Hex()) {
		fmt.Println("Pending owner does NOT match this key — aborting.")
		return
	}

	// Contract addresses transferred by TransferOwnership.s.sol — pulled from env.
	// Replace with hard-coded addresses from your broadcast/ folder if preferred.
	targets := []target{
		{"RouteRegistry", common.HexToAddress(envOr("ROUTE_REGISTRY", ""))},
		{"Afi", common.HexToAddress(envOr("AFI", ""))},
		{"CakeV3", common.HexToAddress(envOr("CAKEV3", ""))},
		{"UniV3", common.HexToAddress(envOr("UNIV3", ""))},
		{"UniV4", common.HexToAddress(envOr("UNIV4", ""))},
		{"Aerodrome", common.HexToAddress(envOr("AERODROME", ""))},
		{"BalancerV3", common.HexToAddress(envOr("BALANCERV3", ""))},
		{"Curve128", common.HexToAddress(envOr("CURVE128", ""))},
		{"Curve256", common.HexToAddress(envOr("CURVE256", ""))},
		{"FluidDex", common.HexToAddress(envOr("FLUIDDEX", ""))},
		{"AaveLiquidator", common.HexToAddress(envOr("AAVE_LIQUIDATOR", ""))},
	}

	parsed, err := abi.JSON(strings.NewReader(ownable2StepABI))
	if err != nil {
		log.Fatal("parse ABI:", err)
	}
	calldata, err := parsed.Pack("acceptOwnership")
	if err != nil {
		log.Fatal("pack:", err)
	}

	// Step 2: call acceptOwnership() on each non-empty target.
	// TODO: client.AcceptOwnership / AcceptOwnershipBatch when SDK lands them.
	fmt.Println("\nAccepting ownership on 12 contracts...")
	for _, t := range targets {
		if t.addr == (common.Address{}) {
			fmt.Printf("  [skip] %-15s (env not set)\n", t.name)
			continue
		}
		txHash, err := sendRawTx(ctx, rpcURL, ownerKey, t.addr, calldata)
		if err != nil {
			log.Fatalf("acceptOwnership(%s): %v", t.name, err)
		}
		fmt.Printf("  [accept] %-15s → %s\n", t.name, txHash)
		if _, err := client.WaitForTx(ctx, txHash); err != nil {
			log.Fatal("wait:", err)
		}
	}

	// Final sanity: AFI router pendingOwner should now be address(0).
	after, err := client.GetPendingOwner(ctx, afiAddr, afi.BaseChainID)
	if err != nil {
		log.Fatal("getPendingOwner after:", err)
	}
	fmt.Printf("\nAFI router pendingOwner() after: %s\n", after.Hex())
	fmt.Println("Ownership handover complete.")
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
		log.Fatalf("env %s is required", k)
	}
	return v
}
