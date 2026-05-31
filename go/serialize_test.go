package afi

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func sampleQuote() *Quote {
	return &Quote{
		TokenIn:      common.HexToAddress("0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"),
		TokenOut:     common.HexToAddress("0x4200000000000000000000000000000000000006"),
		AmountIn:     "1000",
		AmountOut:    "0.5",
		MinOut:       "0.495",
		AmountInWei:  big.NewInt(1_000_000_000),
		AmountOutWei: new(big.Int).SetUint64(500_000_000_000_000_000),
		MinOutWei:    new(big.Int).SetUint64(495_000_000_000_000_000),
		Steps:        []byte{0xde, 0xad, 0xbe, 0xef},
		Path: []common.Address{
			common.HexToAddress("0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"),
			common.HexToAddress("0x4200000000000000000000000000000000000006"),
		},
		Hops: []Hop{{
			TokenIn:      common.HexToAddress("0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"),
			TokenOut:     common.HexToAddress("0x4200000000000000000000000000000000000006"),
			AmountIn:     "1000",
			AmountOut:    "0.5",
			MinOut:       "0.495",
			AmountInWei:  big.NewInt(1_000_000_000),
			AmountOutWei: new(big.Int).SetUint64(500_000_000_000_000_000),
			MinOutWei:    new(big.Int).SetUint64(495_000_000_000_000_000),
			TokenInPrice: "1", TokenOutPrice: "1",
			Slippage: 0.5, Type: "v3", Kind: "uniswap", RouteID: 1, Weight: 1.0,
		}},
		Slippage:      0.5,
		FeeBps:        35,
		TokenInPrice:  "1",
		TokenOutPrice: "1",
		CreatedAt:     1_700_000_000_000,
	}
}

func TestQuote_MarshalJSON_RoundTrip(t *testing.T) {
	q := sampleQuote()
	data, err := json.Marshal(q)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Wei fields should be quoted strings, not raw numbers
	if want := `"amountInWei":"1000000000"`; !contains(string(data), want) {
		t.Errorf("expected %q in JSON, got: %s", want, data)
	}

	var restored Quote
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if restored.AmountInWei.Cmp(q.AmountInWei) != 0 {
		t.Errorf("AmountInWei mismatch: %s vs %s", restored.AmountInWei, q.AmountInWei)
	}
	if restored.MinOutWei.Cmp(q.MinOutWei) != 0 {
		t.Errorf("MinOutWei mismatch: %s vs %s", restored.MinOutWei, q.MinOutWei)
	}
	if restored.Hops[0].AmountOutWei.Cmp(q.Hops[0].AmountOutWei) != 0 {
		t.Error("Hop amountOutWei mismatch")
	}
	if restored.CreatedAt != q.CreatedAt {
		t.Errorf("CreatedAt mismatch: %d vs %d", restored.CreatedAt, q.CreatedAt)
	}
}

func TestSwapResult_MarshalJSON_RoundTrip(t *testing.T) {
	r := &SwapResult{
		TxHash:      common.HexToHash("0xabc"),
		BlockNumber: 1234567890123,
		AmountIn:    big.NewInt(1_000_000_000),
		AmountOut:   new(big.Int).SetUint64(500_000_000_000_000_000),
		TokenIn:     common.HexToAddress("0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"),
		TokenOut:    common.HexToAddress("0x4200000000000000000000000000000000000006"),
		GasUsed:     150_000,
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !contains(string(data), `"amountIn":"1000000000"`) {
		t.Errorf("amountIn should be string-encoded: %s", data)
	}

	var restored SwapResult
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if restored.AmountIn.Cmp(r.AmountIn) != 0 {
		t.Error("AmountIn mismatch")
	}
	if restored.BlockNumber != r.BlockNumber {
		t.Error("BlockNumber mismatch")
	}
}

func TestTokenInfo_MarshalJSON_WithBalance(t *testing.T) {
	info := &TokenInfo{
		Address:   common.HexToAddress("0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"),
		Symbol:    "USDC",
		Name:      "USD Coin",
		Decimals:  6,
		Owner:     common.HexToAddress("0x1234567890123456789012345678901234567890"),
		Balance:   big.NewInt(1_000_000),
		Allowance: big.NewInt(500_000),
	}
	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !contains(string(data), `"balance":"1000000"`) {
		t.Errorf("balance string-encoded missing: %s", data)
	}

	var restored TokenInfo
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if restored.Balance.Cmp(info.Balance) != 0 {
		t.Error("Balance mismatch")
	}
}

func TestTokenInfo_MetadataOnly(t *testing.T) {
	info := &TokenInfo{
		Address:  common.HexToAddress("0xabc"),
		Symbol:   "X",
		Name:     "X",
		Decimals: 18,
	}
	data, _ := json.Marshal(info)
	var restored TokenInfo
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if restored.Balance != nil {
		t.Errorf("Balance should be nil, got %v", restored.Balance)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
