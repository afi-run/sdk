package afi

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestEncodeNMRRequestOperation(t *testing.T) {
	data, err := EncodeNMRRequestOperation(
		common.HexToAddress("0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"),
		big.NewInt(1_000_000),
		[]byte{0xde, 0xad},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) < 4 {
		t.Fatalf("calldata too short: %d", len(data))
	}
	method := nmrParsedABI.Methods["requestOperation"]
	if string(data[:4]) != string(method.ID) {
		t.Error("selector mismatch for requestOperation")
	}
}

func TestEncodeNMRSwap(t *testing.T) {
	data, err := EncodeNMRSwap(
		common.HexToAddress("0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"),
		big.NewInt(1_000_000),
		big.NewInt(1_010_000),
		[]byte{0x01},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) < 4 {
		t.Fatalf("calldata too short: %d", len(data))
	}
	method := nmrParsedABI.Methods["swap"]
	if string(data[:4]) != string(method.ID) {
		t.Error("selector mismatch for swap")
	}
}

func TestEncodeNMRLoan(t *testing.T) {
	data, err := EncodeNMRLoan(
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		common.HexToAddress("0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"),
		big.NewInt(1_000_000),
		big.NewInt(1_010_000),
		[]byte{0x02},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) < 4 {
		t.Fatalf("calldata too short: %d", len(data))
	}
	method := nmrParsedABI.Methods["loan"]
	if string(data[:4]) != string(method.ID) {
		t.Error("selector mismatch for loan")
	}
}

func TestEncodeNMRSweepProfit(t *testing.T) {
	data, err := EncodeNMRSweepProfit(
		common.HexToAddress("0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"),
		big.NewInt(1_000_000),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) < 4 {
		t.Fatalf("calldata too short: %d", len(data))
	}
	method := nmrParsedABI.Methods["sweepProfit"]
	if string(data[:4]) != string(method.ID) {
		t.Error("selector mismatch for sweepProfit")
	}
}

func TestEncodeNMRSetTreasury(t *testing.T) {
	data, err := EncodeNMRSetTreasury(common.HexToAddress("0x2222222222222222222222222222222222222222"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) < 4 {
		t.Fatalf("calldata too short: %d", len(data))
	}
	method := nmrParsedABI.Methods["setTreasury"]
	if string(data[:4]) != string(method.ID) {
		t.Error("selector mismatch for setTreasury")
	}
}
