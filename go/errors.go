package afi

import (
	"fmt"
	"math/big"
)

type AfiError struct {
	Code    string
	Message string

	// Optional context, populated for specific Codes:
	// - INSUFFICIENT_BALANCE: Token, Owner, Symbol, Decimals, Balance, Required
	Token    string
	Owner    string
	Symbol   string
	Decimals uint8
	Balance  *big.Int
	Required *big.Int
	// Decoded carries the structured revert reason when DecodeRevertReason
	// matched the raw data. Populated for SIMULATION_FAILED / SWAP_REVERTED codes.
	Decoded *DecodedRevert
}

func (e *AfiError) Error() string { return e.Message }

func newErr(code, format string, args ...any) *AfiError {
	return &AfiError{Code: code, Message: fmt.Sprintf(format, args...)}
}

// formatWei formats a wei amount using `decimals`. When decimals == 0 returns the raw string.
func formatWei(amount *big.Int, decimals uint8) string {
	if decimals == 0 {
		return amount.String()
	}
	return formatAmount(amount, decimals)
}

// ErrInsufficientBalance builds the error message and (when known) attaches symbol/decimals
// for a much friendlier display ("Insufficient USDC: have 0.5, need 1") instead of raw addresses.
func ErrInsufficientBalance(token string, balance, required *big.Int) error {
	return ErrInsufficientBalanceDetailed(token, "", "", 0, balance, required)
}

// ErrInsufficientBalanceDetailed lets callers pass the resolved owner, symbol and decimals
// for a richer error message and structured fields.
func ErrInsufficientBalanceDetailed(token, owner, symbol string, decimals uint8, balance, required *big.Int) error {
	label := token
	if symbol != "" {
		label = symbol
	}
	have := formatWei(balance, decimals)
	need := formatWei(required, decimals)
	ownerPart := ""
	if owner != "" {
		ownerPart = " for " + owner
	}
	return &AfiError{
		Code:     "INSUFFICIENT_BALANCE",
		Message:  fmt.Sprintf("insufficient %s%s: have %s, need %s", label, ownerPart, have, need),
		Token:    token,
		Owner:    owner,
		Symbol:   symbol,
		Decimals: decimals,
		Balance:  balance,
		Required: required,
	}
}

func ErrQuote(reason string) error {
	return newErr("QUOTE_FAILED", "quote failed: %s", reason)
}

func ErrSimulation(reason string) error {
	return newErr("SIMULATION_FAILED", "swap simulation reverted: %s", reason)
}

// ErrSimulationDecoded is like ErrSimulation but attaches the structured DecodedRevert.
func ErrSimulationDecoded(reason string, decoded *DecodedRevert) error {
	e := &AfiError{
		Code:    "SIMULATION_FAILED",
		Message: fmt.Sprintf("swap simulation reverted: %s", reason),
		Decoded: decoded,
	}
	return e
}

func ErrApproval(reason string) error {
	return newErr("APPROVAL_FAILED", "token approval failed: %s", reason)
}

func ErrSwapReverted(reason string) error {
	return newErr("SWAP_REVERTED", "swap reverted: %s", reason)
}

// ErrSwapRevertedDecoded is like ErrSwapReverted but attaches the structured DecodedRevert.
func ErrSwapRevertedDecoded(reason string, decoded *DecodedRevert) error {
	return &AfiError{
		Code:    "SWAP_REVERTED",
		Message: fmt.Sprintf("swap reverted: %s", reason),
		Decoded: decoded,
	}
}

func ErrNoSigner() error {
	return newErr("NO_SIGNER", "private key required — create the client with PrivateKey set or call Connect()")
}
