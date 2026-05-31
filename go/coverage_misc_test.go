package afi

import (
	"encoding/json"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestSendOptions_Apply(t *testing.T) {
	var o sendOptions
	for _, opt := range []SendOption{
		WithValue(big.NewInt(5)),
		WithConfirmations(3),
		WithTimeoutMs(1234),
		WithNonce(7),
		WithGasBuffer(20),
		WithoutAllowancePrecheck(),
	} {
		opt(&o)
	}
	if o.value.Cmp(big.NewInt(5)) != 0 || o.confirmations != 3 || o.timeoutMs != 1234 {
		t.Errorf("scalar opts not applied: %+v", o)
	}
	if o.nonce == nil || *o.nonce != 7 || o.gasBuffer == nil || *o.gasBuffer != 20 || !o.skipAllowanceCheck {
		t.Errorf("pointer opts not applied: %+v", o)
	}
}

func TestPrecheckEnabled(t *testing.T) {
	if !precheckEnabled(nil) {
		t.Error("precheck should be enabled by default")
	}
	if precheckEnabled([]SendOption{WithoutAllowancePrecheck()}) {
		t.Error("precheck should be disabled by WithoutAllowancePrecheck")
	}
}

func TestErrorTypes_Messages(t *testing.T) {
	if (&AfiError{Message: "x"}).Error() != "x" {
		t.Error("AfiError.Error")
	}
	if (&BadRequestError{Status: 400, Body: "bad"}).Error() == "" {
		t.Error("BadRequestError.Error")
	}
	if (&ServerError{Status: 500, Body: "boom"}).Error() == "" {
		t.Error("ServerError.Error")
	}
	wrapped := errors.New("dial fail")
	ne := &NetworkError{Err: wrapped}
	if ne.Error() == "" {
		t.Error("NetworkError.Error")
	}
	if ne.Unwrap() != wrapped {
		t.Error("NetworkError.Unwrap")
	}
	if err := ErrInsufficientAllowance("0xtok", "0xown", "0xspend", big.NewInt(1), big.NewInt(10)); err == nil || err.Error() == "" {
		t.Error("ErrInsufficientAllowance")
	}
}

func TestEncoders(t *testing.T) {
	tok := common.HexToAddress("0x1111111111111111111111111111111111111111")
	q := &Quote{
		TokenIn:     tok,
		TokenOut:    tok,
		AmountInWei: big.NewInt(1000),
		MinOutWei:   big.NewInt(900),
		Steps:       []byte{0x01},
	}
	if tx, err := EncodeSwap(q); err != nil || tx == nil {
		t.Errorf("EncodeSwap: %v", err)
	}
	if tx, err := EncodeApprove(tok, big.NewInt(1000)); err != nil || tx == nil {
		t.Errorf("EncodeApprove: %v", err)
	}
	if tx, err := EncodeRevoke(tok); err != nil || tx == nil {
		t.Errorf("EncodeRevoke: %v", err)
	}
}

func TestAccountOverride_MarshalJSON(t *testing.T) {
	a := AccountOverride{StateDiff: map[common.Hash]common.Hash{
		common.HexToHash("0x01"): common.HexToHash("0x02"),
	}}
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if len(b) == 0 || string(b) == "{}" {
		t.Errorf("unexpected marshal output: %s", b)
	}
	if _, err := json.Marshal(AccountOverride{}); err != nil {
		t.Errorf("marshal empty: %v", err)
	}
}

func TestValidateUint160(t *testing.T) {
	if err := validateUint160("x", big.NewInt(1)); err != nil {
		t.Errorf("valid value: %v", err)
	}
	if err := validateUint160("x", nil); err != nil {
		t.Errorf("nil value: %v", err)
	}
	over := new(big.Int).Lsh(big.NewInt(1), 160) // 2^160 — out of uint160 range
	if err := validateUint160("x", over); err == nil {
		t.Error("expected overflow error for 2^160")
	}
}
