// Example 8: NMR user-funded arbitrage via NMR.loan(...)
//
// Unlike requestOperation (flash-loan funded), `loan(user, asset, amount, minOut, params)`
// uses the *user's* tokens. The user must approve NMR for `amount` of `asset` first —
// the operator can NOT bypass this. NMR pulls the funds, runs the cycle, returns
// `out >= minOut` to the user, and keeps the operator share.
//
// Required approval (run from the user's wallet before the operator submits):
//
//	IERC20(asset).approve(NMR_ADDRESS, amount)
//
// This example performs that approval explicitly first.
//
// Prerequisites:
//   - RPC_URL                — Base RPC endpoint
//   - USER_PRIVATE_KEY       — funds the trade (must hold `amount` of `asset`)
//   - OPERATOR_PRIVATE_KEY   — submits the loan tx (one of the 7 registered ops)
//
// NOTE: high-level `client.NMRLoanArbitrage()` is not yet exposed.
// TODO: client.NMRLoanArbitrage when SDK lands it.
//
// Run: go run ./examples/go/nmr-loan
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

func main() {
	rpcURL := envOr("RPC_URL", "https://mainnet.base.org")
	userKey := mustEnv("USER_PRIVATE_KEY")
	opKey := mustEnv("OPERATOR_PRIVATE_KEY")

	client, err := afi.NewClient(afi.Config{RPCURL: rpcURL, PrivateKey: opKey})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()
	usdc := common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913")
	nmrAddr := afi.NMRAddresses[afi.NetworkBase]

	userAddr := addressFromKey(userKey)
	fmt.Printf("User:     %s\n", userAddr.Hex())
	fmt.Printf("Operator: %s\n", client.Address().Hex())
	fmt.Printf("NMR:      %s\n\n", nmrAddr.Hex())

	amount := big.NewInt(1_000_000_000) // 1000 USDC
	minOut := big.NewInt(1_005_000_000) // require 0.5% gain

	// Hard-coded route — replace with output of findArbitrage().
	steps := []afi.Step{
		{ID: 3, Data: []byte{}}, // UniV3 placeholder
		{ID: 5, Data: []byte{}}, // Aerodrome placeholder
	}
	params, err := afi.EncodeSteps(steps)
	if err != nil {
		log.Fatal("encode steps:", err)
	}

	// Step 1: user.approve(NMR, amount) — MANDATORY.
	fmt.Println("Step 1: user approves NMR to pull USDC...")
	erc20, err := abi.JSON(strings.NewReader(afi.ERC20ABIJSON))
	if err != nil {
		log.Fatal(err)
	}
	approveData, err := erc20.Pack("approve", nmrAddr, amount)
	if err != nil {
		log.Fatal(err)
	}
	approveHash, err := sendRawTx(ctx, rpcURL, userKey, usdc, approveData)
	if err != nil {
		log.Fatal("approve:", err)
	}
	if _, err := client.WaitForTx(ctx, approveHash); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  approved: %s\n\n", approveHash)

	// Step 2: operator submits NMR.loan(user, asset, amount, minOut, params).
	fmt.Println("Step 2: operator submits NMR.loan(...)")
	calldata, err := afi.EncodeNMRLoan(userAddr, usdc, amount, minOut, params)
	if err != nil {
		log.Fatal("encode loan:", err)
	}
	// TODO: client.NMRLoanArbitrage when SDK lands it.
	loanHash, err := sendRawTx(ctx, rpcURL, opKey, nmrAddr, calldata)
	if err != nil {
		log.Fatal("loan:", err)
	}
	rcpt, err := client.WaitForTx(ctx, loanHash)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  loan tx:  %s\n", loanHash)
	fmt.Printf("  block %d, gas %d\n\n", rcpt.BlockNumber, rcpt.GasUsed)

	eth, err := ethclient.Dial(rpcURL)
	if err != nil {
		log.Fatal(err)
	}
	defer eth.Close()
	r, err := eth.TransactionReceipt(ctx, common.HexToHash(loanHash))
	if err != nil {
		log.Fatal(err)
	}
	for _, ev := range mustExecuted(afi.ParseFlashLoanExecuted(r.Logs)) {
		fmt.Println("FlashLoanExecuted:")
		fmt.Printf("  amount:  %s USDC\n", afi.FormatUnits(ev.Amount, 6))
		fmt.Printf("  premium: %s USDC\n", afi.FormatUnits(ev.Premium, 6))
		fmt.Printf("  profit:  %s USDC\n", afi.FormatUnits(ev.Profit, 6))
	}
}

func mustExecuted(evs []afi.FlashLoanExecutedEvent, err error) []afi.FlashLoanExecutedEvent {
	if err != nil {
		log.Fatal(err)
	}
	return evs
}

func addressFromKey(keyHex string) common.Address {
	k, err := crypto.HexToECDSA(strings.TrimPrefix(keyHex, "0x"))
	if err != nil {
		log.Fatal("parse key:", err)
	}
	return crypto.PubkeyToAddress(k.PublicKey)
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
