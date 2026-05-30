package afi

import (
	"errors"
	"math/big"
	"testing"
)

func TestErrInsufficientBalance(t *testing.T) {
	err := ErrInsufficientBalance("0xabc", big.NewInt(100), big.NewInt(500))

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}

	var afiErr *AfiError
	if !errors.As(err, &afiErr) {
		t.Fatal("error should be *AfiError")
	}
	if afiErr.Code != "INSUFFICIENT_BALANCE" {
		t.Errorf("Code = %q, want INSUFFICIENT_BALANCE", afiErr.Code)
	}
	if afiErr.Message == "" {
		t.Error("Message should not be empty")
	}
	if afiErr.Balance.Cmp(big.NewInt(100)) != 0 || afiErr.Required.Cmp(big.NewInt(500)) != 0 {
		t.Errorf("Balance/Required not attached: %+v", afiErr)
	}
}

func TestErrInsufficientBalanceDetailed(t *testing.T) {
	err := ErrInsufficientBalanceDetailed("0xabc", "0xOwner", "USDC", 6, big.NewInt(500_000), big.NewInt(1_000_000))

	var afiErr *AfiError
	if !errors.As(err, &afiErr) {
		t.Fatal("expected *AfiError")
	}
	msg := afiErr.Error()
	for _, want := range []string{"USDC", "0xOwner", "0.5", "1"} {
		if !contains(msg, want) {
			t.Errorf("msg should contain %q: %s", want, msg)
		}
	}
	if afiErr.Symbol != "USDC" || afiErr.Decimals != 6 || afiErr.Owner != "0xOwner" {
		t.Errorf("context fields not populated: %+v", afiErr)
	}
}

func TestErrQuote(t *testing.T) {
	err := ErrQuote("no route found")

	var afiErr *AfiError
	if !errors.As(err, &afiErr) {
		t.Fatal("error should be *AfiError")
	}
	if afiErr.Code != "QUOTE_FAILED" {
		t.Errorf("Code = %q, want QUOTE_FAILED", afiErr.Code)
	}
}

func TestErrSimulation(t *testing.T) {
	err := ErrSimulation("minOut not met")

	var afiErr *AfiError
	if !errors.As(err, &afiErr) {
		t.Fatal("error should be *AfiError")
	}
	if afiErr.Code != "SIMULATION_FAILED" {
		t.Errorf("Code = %q, want SIMULATION_FAILED", afiErr.Code)
	}
}

func TestErrApproval(t *testing.T) {
	err := ErrApproval("user rejected")

	var afiErr *AfiError
	if !errors.As(err, &afiErr) {
		t.Fatal("error should be *AfiError")
	}
	if afiErr.Code != "APPROVAL_FAILED" {
		t.Errorf("Code = %q, want APPROVAL_FAILED", afiErr.Code)
	}
}

func TestErrSwapReverted(t *testing.T) {
	err := ErrSwapReverted("execution reverted")

	var afiErr *AfiError
	if !errors.As(err, &afiErr) {
		t.Fatal("error should be *AfiError")
	}
	if afiErr.Code != "SWAP_REVERTED" {
		t.Errorf("Code = %q, want SWAP_REVERTED", afiErr.Code)
	}
}

func TestErrNoSigner(t *testing.T) {
	err := ErrNoSigner()

	var afiErr *AfiError
	if !errors.As(err, &afiErr) {
		t.Fatal("error should be *AfiError")
	}
	if afiErr.Code != "NO_SIGNER" {
		t.Errorf("Code = %q, want NO_SIGNER", afiErr.Code)
	}
	if afiErr.Message == "" {
		t.Error("Message should not be empty")
	}
}

func TestAfiErrorInterface(t *testing.T) {
	err := ErrQuote("test")
	// AfiError must implement the error interface
	var _ error = err
	// Must be unwrappable with errors.As
	var afiErr *AfiError
	if !errors.As(err, &afiErr) {
		t.Error("errors.As should work with *AfiError")
	}
}
