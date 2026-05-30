package afi

import (
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func encodeRevert(t *testing.T, signature string, paramTypes []string, values []interface{}) []byte {
	t.Helper()
	hash := crypto.Keccak256([]byte(signature))
	selector := hash[:4]
	if len(paramTypes) == 0 {
		return selector
	}
	args := abi.Arguments{}
	for _, ts := range paramTypes {
		ty, err := abi.NewType(ts, "", nil)
		if err != nil {
			t.Fatalf("type %q: %v", ts, err)
		}
		args = append(args, abi.Argument{Type: ty})
	}
	payload, err := args.Pack(values...)
	if err != nil {
		t.Fatalf("pack: %v", err)
	}
	return append(selector, payload...)
}

func TestDecodeRevertReason_BuiltInErrorString(t *testing.T) {
	data := encodeRevert(t, "Error(string)", []string{"string"}, []interface{}{"boom"})
	r := DecodeRevertReason(data)
	if r == nil {
		t.Fatal("expected non-nil")
	}
	if r.Name != "Error" {
		t.Errorf("Name = %q", r.Name)
	}
	if r.Args[0].(string) != "boom" {
		t.Errorf("arg mismatch: %v", r.Args[0])
	}
}

func TestDecodeRevertReason_BuiltInPanic(t *testing.T) {
	data := encodeRevert(t, "Panic(uint256)", []string{"uint256"}, []interface{}{big.NewInt(0x11)})
	r := DecodeRevertReason(data)
	if r == nil || r.Name != "Panic" {
		t.Fatalf("expected Panic, got %+v", r)
	}
}

func TestDecodeRevertReason_AFIInsufficientFunds(t *testing.T) {
	data := encodeRevert(t, "InsufficientFunds(uint256)", []string{"uint256"}, []interface{}{big.NewInt(1234)})
	r := DecodeRevertReason(data)
	if r == nil {
		t.Fatal("expected non-nil")
	}
	if r.Name != "InsufficientFunds" {
		t.Errorf("Name = %q", r.Name)
	}
	if r.Args[0].(*big.Int).Cmp(big.NewInt(1234)) != 0 {
		t.Errorf("arg mismatch: %v", r.Args[0])
	}
}

func TestDecodeRevertReason_AFIZeroAddress(t *testing.T) {
	data := encodeRevert(t, "ZeroAddress()", []string{}, []interface{}{})
	r := DecodeRevertReason(data)
	if r == nil || r.Name != "ZeroAddress" {
		t.Fatalf("got %+v", r)
	}
}

func TestDecodeRevertReason_AFIDifferentAssets(t *testing.T) {
	expected := common.HexToAddress("0x4200000000000000000000000000000000000006")
	actual := common.HexToAddress("0x833589fcd6edb6e08f4c7c32d4f71b54bda02913")
	data := encodeRevert(t, "DifferentAssets(address,address)", []string{"address", "address"},
		[]interface{}{expected, actual})
	r := DecodeRevertReason(data)
	if r == nil {
		t.Fatal("expected non-nil")
	}
	if r.Name != "DifferentAssets" {
		t.Errorf("Name = %q", r.Name)
	}
	if r.Args[0].(common.Address) != expected {
		t.Errorf("expected mismatch: %v", r.Args[0])
	}
}

func TestDecodeRevertReason_OZErrors(t *testing.T) {
	cases := []string{
		"OwnableInvalidOwner(address)",
		"OwnableUnauthorizedAccount(address)",
		"ReentrancyGuardReentrantCall()",
	}
	for _, sig := range cases {
		t.Run(sig, func(t *testing.T) {
			var types []string
			var values []interface{}
			if strings.Contains(sig, "address") {
				types = []string{"address"}
				values = []interface{}{common.HexToAddress("0xdeadbeef00000000000000000000000000000000")}
			}
			data := encodeRevert(t, sig, types, values)
			r := DecodeRevertReason(data)
			if r == nil {
				t.Fatalf("expected non-nil for %s", sig)
			}
		})
	}
}

func TestDecodeRevertReason_EmptyOrShort(t *testing.T) {
	if DecodeRevertReason(nil) != nil || DecodeRevertReason([]byte{0x01, 0x02}) != nil {
		t.Error("expected nil for empty/short data")
	}
}

func TestDecodeRevertReason_UnknownSelector(t *testing.T) {
	data := encodeRevert(t, "Mystery(uint256)", []string{"uint256"}, []interface{}{big.NewInt(1)})
	if DecodeRevertReason(data) != nil {
		t.Error("expected nil for unknown selector")
	}
}

func TestRegisterCustomErrors_DecodesAfterRegistration(t *testing.T) {
	data := encodeRevert(t, "MyContractError(uint256,string)", []string{"uint256", "string"},
		[]interface{}{big.NewInt(42), "details"})

	if DecodeRevertReason(data) != nil {
		t.Fatal("should not decode before registration")
	}

	a, err := abi.JSON(strings.NewReader(`[{"type":"error","name":"MyContractError","inputs":[
		{"name":"code","type":"uint256"},
		{"name":"msg","type":"string"}
	]}]`))
	if err != nil {
		t.Fatal(err)
	}
	RegisterCustomErrors(a)

	r := DecodeRevertReason(data)
	if r == nil || r.Name != "MyContractError" {
		t.Fatalf("expected MyContractError, got %+v", r)
	}
	if r.Args[0].(*big.Int).Cmp(big.NewInt(42)) != 0 {
		t.Errorf("arg 0 mismatch: %v", r.Args[0])
	}
	if r.Args[1].(string) != "details" {
		t.Errorf("arg 1 mismatch: %v", r.Args[1])
	}
}

func TestDecodedRevert_StringFormatting(t *testing.T) {
	d := &DecodedRevert{Name: "InsufficientFunds", Args: []interface{}{big.NewInt(100)}}
	if d.String() != "InsufficientFunds(100)" {
		t.Errorf("got %q", d.String())
	}
	d0 := &DecodedRevert{Name: "ZeroAddress"}
	if d0.String() != "ZeroAddress" {
		t.Errorf("got %q", d0.String())
	}
	var nilD *DecodedRevert
	if nilD.String() != "unknown revert" {
		t.Errorf("got %q", nilD.String())
	}
}

func TestErrSimulationDecoded_Roundtrip(t *testing.T) {
	d := &DecodedRevert{Name: "InsufficientFunds", Args: []interface{}{big.NewInt(100)}}
	err := ErrSimulationDecoded(d.String(), d)
	var afiErr *AfiError
	if !errors.As(err, &afiErr) {
		t.Fatal("expected *AfiError")
	}
	if afiErr.Code != "SIMULATION_FAILED" {
		t.Errorf("Code = %q", afiErr.Code)
	}
	if afiErr.Decoded == nil || afiErr.Decoded.Name != "InsufficientFunds" {
		t.Errorf("Decoded missing: %+v", afiErr.Decoded)
	}
}

func TestGetRegisteredErrorSelectors_IncludesAFIErrors(t *testing.T) {
	selectors := GetRegisteredErrorSelectors()
	if len(selectors) < 9 {
		t.Errorf("expected at least 9 registered selectors (AFI errors), got %d", len(selectors))
	}
}
