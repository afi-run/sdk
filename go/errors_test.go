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
