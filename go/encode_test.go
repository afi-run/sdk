package afi

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

func sampleQuoteForEncode() *Quote {
	return &Quote{
		TokenIn:      common.HexToAddress("0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"),
		TokenOut:     common.HexToAddress("0x4200000000000000000000000000000000000006"),
		AmountInWei:  big.NewInt(1_000_000_000),
		AmountOutWei: new(big.Int).SetUint64(500_000_000_000_000_000),
		MinOutWei:    new(big.Int).SetUint64(495_000_000_000_000_000),
		Steps:        []byte{0xde, 0xad, 0xbe, 0xef},
	}
}

func TestEncodeSwap_TargetsRouter(t *testing.T) {
	tx, err := EncodeSwap(sampleQuoteForEncode())
	if err != nil {
		t.Fatal(err)
	}
	if tx.To != AfiAddress {
		t.Errorf("To = %s, want AfiAddress", tx.To.Hex())
	}
	if tx.Value.Sign() != 0 {
		t.Errorf("Value should be 0, got %s", tx.Value)
	}
	if len(tx.Data) == 0 {
		t.Error("Data should be non-empty")
	}
	// Verify the calldata decodes back to swap(...) with the expected first arg.
	method := afiABICached.Methods["swap"]
	if string(tx.Data[:4]) != string(method.ID) {
		t.Error("calldata selector mismatch")
	}
	args, err := method.Inputs.Unpack(tx.Data[4:])
	if err != nil {
		t.Fatalf("unpack: %v", err)
	}
	if args[0].(common.Address) != sampleQuoteForEncode().TokenIn {
		t.Error("decoded tokenIn mismatch")
	}
}

func TestEncodeApprove_TargetsToken(t *testing.T) {
	token := common.HexToAddress("0x833589fcd6edb6e08f4c7c32d4f71b54bda02913")
	tx, err := EncodeApprove(token, big.NewInt(1_000_000))
	if err != nil {
		t.Fatal(err)
	}
	if tx.To != token {
		t.Errorf("To = %s, want token", tx.To.Hex())
	}
	method := erc20ABICached.Methods["approve"]
	if string(tx.Data[:4]) != string(method.ID) {
		t.Error("calldata selector mismatch")
	}
	args, err := method.Inputs.Unpack(tx.Data[4:])
	if err != nil {
		t.Fatalf("unpack: %v", err)
	}
	if args[0].(common.Address) != AfiAddress {
		t.Error("decoded spender should be AfiAddress")
	}
	if args[1].(*big.Int).Cmp(big.NewInt(1_000_000)) != 0 {
		t.Errorf("decoded amount mismatch: %s", args[1])
	}
}

func TestEncodeRevoke_AmountIsZero(t *testing.T) {
	token := common.HexToAddress("0x833589fcd6edb6e08f4c7c32d4f71b54bda02913")
	tx, err := EncodeRevoke(token)
	if err != nil {
		t.Fatal(err)
	}
	method := erc20ABICached.Methods["approve"]
	args, _ := method.Inputs.Unpack(tx.Data[4:])
	if args[1].(*big.Int).Sign() != 0 {
		t.Errorf("revoke amount should be 0, got %s", args[1])
	}
}

func TestERC20ABICached_HasApprove(t *testing.T) {
	if _, ok := erc20ABICached.Methods["approve"]; !ok {
		t.Error("expected approve in cached ERC20 ABI")
	}
	_ = abi.ABI{} // silence unused import warning
}
