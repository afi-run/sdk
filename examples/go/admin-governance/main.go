// Example 9: Owner-only admin governance flow
//
// Reads the Afi router state, then exercises three owner-restricted operations:
//  1. adminPause()           — halt all swap/swapFor traffic
//  2. adminSetFeeBps(25)     — change the default protocol fee
//  3. adminAddRule(rule)     — append a custom routing rule (optional)
//
// WARNING: only the contract owner can run this. OWNER_PRIVATE_KEY must hold the
// key for the address returned by Afi.owner(). With a wrong key every tx reverts
// with OwnableUnauthorizedAccount.
//
// Prerequisites:
//   - RPC_URL              — Base RPC endpoint
//   - OWNER_PRIVATE_KEY    — Afi contract owner (Base)
//   - RULE_ADDR (optional) — address of a deployed IRule to add
//
// NOTE: convenience wrappers `client.AdminPause()` etc. are not exposed yet.
// This uses afi.EncodeAfi* + raw ethclient broadcast.
// TODO: client.Admin* when SDK lands it.
//
// Run: go run ./examples/go/admin-governance
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
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	rpcURL := envOr("RPC_URL", "https://mainnet.base.org")
	ownerKey := mustEnv("OWNER_PRIVATE_KEY")
	ruleAddr := common.HexToAddress(envOr("RULE_ADDR", "0x0000000000000000000000000000000000000000"))
	sampleUser := common.HexToAddress("0x1111111111111111111111111111111111111111")

	client, err := afi.NewClient(afi.Config{RPCURL: rpcURL, PrivateKey: ownerKey})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()
	afiAddr := afi.AfiAddresses[afi.NetworkBase]
	fmt.Printf("Owner: %s\nAFI:   %s\n\n", client.Address().Hex(), afiAddr.Hex())

	// Pre-state via direct contract read.
	eth, err := ethclient.Dial(rpcURL)
	if err != nil {
		log.Fatal(err)
	}
	defer eth.Close()
	parsed, err := abi.JSON(strings.NewReader(afi.AfiABI))
	if err != nil {
		log.Fatal(err)
	}
	bound := bind.NewBoundContract(afiAddr, parsed, eth, eth, eth)
	callOpts := &bind.CallOpts{Context: ctx}

	paused0 := readBool(bound, callOpts, "paused")
	fee0 := readUint16(bound, callOpts, "feeBps")
	userFee := readUint16(bound, callOpts, "feeBpsOf", sampleUser)
	fmt.Printf("Pre-state:\n")
	fmt.Printf("  paused:         %v\n", paused0)
	fmt.Printf("  default feeBps: %d\n", fee0)
	fmt.Printf("  user feeBps:    %d\n\n", userFee)

	// 1. adminPause.
	// TODO: client.AdminPause when SDK lands it.
	pauseData, err := afi.EncodeAfiPause()
	if err != nil {
		log.Fatal(err)
	}
	h1, err := sendRawTx(ctx, rpcURL, ownerKey, afiAddr, pauseData)
	if err != nil {
		log.Fatal("pause:", err)
	}
	fmt.Printf("adminPause()        → %s\n", h1)
	if _, err := client.WaitForTx(ctx, h1); err != nil {
		log.Fatal(err)
	}

	// 2. adminSetFeeBps(25).
	// TODO: client.AdminSetFeeBps when SDK lands it.
	feeData, err := afi.EncodeAfiSetFeeBps(25)
	if err != nil {
		log.Fatal(err)
	}
	h2, err := sendRawTx(ctx, rpcURL, ownerKey, afiAddr, feeData)
	if err != nil {
		log.Fatal("setFeeBps:", err)
	}
	fmt.Printf("adminSetFeeBps(25)  → %s\n", h2)
	if _, err := client.WaitForTx(ctx, h2); err != nil {
		log.Fatal(err)
	}

	// 3. adminAddRule(ruleAddr) — only if RULE_ADDR was set.
	if ruleAddr != (common.Address{}) {
		// TODO: client.AdminAddRule when SDK lands it.
		ruleData, err := afi.EncodeAfiAddRule(ruleAddr)
		if err != nil {
			log.Fatal(err)
		}
		h3, err := sendRawTx(ctx, rpcURL, ownerKey, afiAddr, ruleData)
		if err != nil {
			log.Fatal("addRule:", err)
		}
		fmt.Printf("adminAddRule(%s) → %s\n", ruleAddr.Hex(), h3)
		if _, err := client.WaitForTx(ctx, h3); err != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Println("adminAddRule()       skipped — set RULE_ADDR to attempt it")
	}

	// Post-state.
	paused1 := readBool(bound, callOpts, "paused")
	fee1 := readUint16(bound, callOpts, "feeBps")
	fmt.Printf("\nPost-state:\n")
	fmt.Printf("  paused:         %v\n", paused1)
	fmt.Printf("  default feeBps: %d\n", fee1)
}

func readBool(b *bind.BoundContract, opts *bind.CallOpts, method string, args ...interface{}) bool {
	var out []interface{}
	if err := b.Call(opts, &out, method, args...); err != nil {
		log.Fatalf("%s: %v", method, err)
	}
	return out[0].(bool)
}

func readUint16(b *bind.BoundContract, opts *bind.CallOpts, method string, args ...interface{}) uint16 {
	var out []interface{}
	if err := b.Call(opts, &out, method, args...); err != nil {
		log.Fatalf("%s: %v", method, err)
	}
	return out[0].(uint16)
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
