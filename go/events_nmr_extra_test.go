package afi

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestParseFlashLoanFailedWithData(t *testing.T) {
	asset := common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913")
	data := []byte{0x08, 0xc3, 0x79, 0xa0}
	log := synthLog(t, nmrParsedABI, common.Address{}, "FlashLoanFailedWithData",
		[]interface{}{asset}, big.NewInt(1000), data)

	out, err := ParseFlashLoanFailedWithData([]*types.Log{log})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("got %d events, want 1", len(out))
	}
	if out[0].Asset != asset {
		t.Errorf("asset = %s, want %s", out[0].Asset, asset)
	}
	if out[0].Amount.String() != "1000" {
		t.Errorf("amount = %s, want 1000", out[0].Amount)
	}
	if string(out[0].Data) != string(data) {
		t.Errorf("data = %x, want %x", out[0].Data, data)
	}
}

func TestParseNMRSwapExecuted(t *testing.T) {
	in := common.HexToAddress("0x0000000000000000000000000000000000000011")
	out := common.HexToAddress("0x0000000000000000000000000000000000000022")
	log := synthLog(t, nmrParsedABI, common.Address{}, "SwapExecuted",
		[]interface{}{in, out}, big.NewInt(500), big.NewInt(525))

	got, err := ParseNMRSwapExecuted([]*types.Log{log})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d events, want 1", len(got))
	}
	if got[0].AssetIn != in || got[0].AssetOut != out {
		t.Errorf("assets = %s/%s, want %s/%s", got[0].AssetIn, got[0].AssetOut, in, out)
	}
	if got[0].AmountIn.String() != "500" || got[0].AmountOut.String() != "525" {
		t.Errorf("amounts = %s/%s, want 500/525", got[0].AmountIn, got[0].AmountOut)
	}
}

// NMR's SwapExecuted must NOT collide with Afi's (different topic0).
func TestNMRSwapExecuted_DistinctFromAfi(t *testing.T) {
	if nmrParsedABI.Events["SwapExecuted"].ID == afiParsedABI.Events["SwapExecuted"].ID {
		t.Error("NMR and Afi SwapExecuted share a topic0 — signatures must differ")
	}
}
