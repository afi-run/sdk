package afi

import (
	"fmt"
	"math/big"
)

type AfiError struct {
	Code    string
	Message string
}

func (e *AfiError) Error() string { return e.Message }

func newErr(code, format string, args ...any) *AfiError {
	return &AfiError{Code: code, Message: fmt.Sprintf(format, args...)}
}

func ErrInsufficientBalance(token string, balance, required *big.Int) error {
	return newErr("INSUFFICIENT_BALANCE",
		"insufficient balance for token %s: have %s, need %s", token, balance, required)
}

func ErrQuote(reason string) error {
	return newErr("QUOTE_FAILED", "quote failed: %s", reason)
}

func ErrSimulation(reason string) error {
	return newErr("SIMULATION_FAILED", "swap simulation reverted: %s", reason)
}

func ErrApproval(reason string) error {
	return newErr("APPROVAL_FAILED", "token approval failed: %s", reason)
}

func ErrSwapReverted(reason string) error {
	return newErr("SWAP_REVERTED", "swap reverted: %s", reason)
}
