package afi

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestParseSwapResult_ReturnsNilWhenNoLog(t *testing.T) {
	receipt := &types.Receipt{
		TxHash:      common.HexToHash("0xabc"),
		BlockNumber: big.NewInt(42),
		GasUsed:     150_000,
		Logs:        []*types.Log{},
	}
	r, err := ParseSwapResult(receipt)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if r != nil {
		t.Errorf("expected nil result for no-log receipt, got %+v", r)
	}
}

func TestParseSwapResult_DecodesSwapExecutedEvent(t *testing.T) {
	from := common.HexToAddress("0x1234567890123456789012345678901234567890")
	assetIn := common.HexToAddress("0x833589fcd6edb6e08f4c7c32d4f71b54bda02913")
	assetOut := common.HexToAddress("0x4200000000000000000000000000000000000006")
	amountIn := big.NewInt(1_000_000)
	amountOut, _ := new(big.Int).SetString("500000000000000000", 10)

	// Build the SwapExecuted log: topics = [event sig, from, assetIn, assetOut], data = abi-encoded (amountIn, amountOut)
	event := afiABICached.Events["SwapExecuted"]
	data, err := event.Inputs.NonIndexed().Pack(amountIn, amountOut)
	if err != nil {
		t.Fatalf("pack data: %v", err)
	}

	log := &types.Log{
		Address: AfiAddress,
		Topics: []common.Hash{
			event.ID,
			common.BytesToHash(from.Bytes()),
			common.BytesToHash(assetIn.Bytes()),
			common.BytesToHash(assetOut.Bytes()),
		},
		Data: data,
	}

	receipt := &types.Receipt{
		TxHash:      common.HexToHash("0xabc"),
		BlockNumber: big.NewInt(42),
		GasUsed:     150_000,
		Logs:        []*types.Log{log},
	}

	r, err := ParseSwapResult(receipt)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if r == nil {
		t.Fatal("expected non-nil result")
	}
	if r.AmountIn.Cmp(amountIn) != 0 {
		t.Errorf("AmountIn = %s, want %s", r.AmountIn, amountIn)
	}
	if r.AmountOut.Cmp(amountOut) != 0 {
		t.Errorf("AmountOut = %s, want %s", r.AmountOut, amountOut)
	}
	if r.TokenIn != assetIn {
		t.Errorf("TokenIn = %s, want %s", r.TokenIn.Hex(), assetIn.Hex())
	}
	if r.TokenOut != assetOut {
		t.Errorf("TokenOut = %s, want %s", r.TokenOut.Hex(), assetOut.Hex())
	}
	if r.BlockNumber != 42 {
		t.Errorf("BlockNumber = %d", r.BlockNumber)
	}
	if r.GasUsed != 150_000 {
		t.Errorf("GasUsed = %d", r.GasUsed)
	}
}

func TestExportedABIJSON(t *testing.T) {
	if AFIABIJSON == "" || ERC20ABIJSON == "" || Multicall3ABIJSON == "" {
		t.Error("ABI JSON exports should be non-empty")
	}
	if !contains(AFIABIJSON, "swap") || !contains(ERC20ABIJSON, "balanceOf") {
		t.Error("ABI JSON contents look wrong")
	}
}
