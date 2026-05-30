package afi

import (
	"encoding/hex"
	"fmt"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
)

// DecodedRevert is a structured revert reason recovered from raw revert data.
type DecodedRevert struct {
	// Name is the error identifier (e.g. "InsufficientFunds").
	Name string
	// Signature is the full Solidity signature ("InsufficientFunds(uint256)").
	Signature string
	// Args are the decoded arguments in declaration order.
	Args []interface{}
}

// String returns a human-friendly summary of the decoded revert.
func (d *DecodedRevert) String() string {
	if d == nil {
		return "unknown revert"
	}
	if len(d.Args) == 0 {
		return d.Name
	}
	parts := make([]string, len(d.Args))
	for i, a := range d.Args {
		parts[i] = fmt.Sprintf("%v", a)
	}
	return fmt.Sprintf("%s(%s)", d.Name, strings.Join(parts, ", "))
}

// Selectors for the built-in Solidity revert variants.
const (
	errorStringSelectorHex = "08c379a0" // Error(string)
	panicSelectorHex       = "4e487b71" // Panic(uint256)
)

var (
	errorRegistryMu sync.RWMutex
	errorRegistry   = map[[4]byte]abi.Error{}

	errorStringDef = mustParseErrorDef(`{"type":"error","name":"Error","inputs":[{"name":"message","type":"string"}]}`)
	panicDef       = mustParseErrorDef(`{"type":"error","name":"Panic","inputs":[{"name":"code","type":"uint256"}]}`)
)

func mustParseErrorDef(jsonDef string) abi.Error {
	a, err := abi.JSON(strings.NewReader("[" + jsonDef + "]"))
	if err != nil {
		panic(fmt.Errorf("parse error def: %w", err))
	}
	for _, e := range a.Errors {
		return e
	}
	panic("no error in def")
}

func registerErrorsFromABI(a abi.ABI) {
	errorRegistryMu.Lock()
	defer errorRegistryMu.Unlock()
	for _, e := range a.Errors {
		var sel [4]byte
		copy(sel[:], e.ID[:4])
		errorRegistry[sel] = e
	}
}

// RegisterCustomErrors adds custom errors from an ABI to the global registry so
// `DecodeRevertReason` can decode them. The AFI router's errors are registered
// automatically at package init.
func RegisterCustomErrors(a abi.ABI) { registerErrorsFromABI(a) }

// GetRegisteredErrorSelectors returns the list of 4-byte selectors currently
// recognised by `DecodeRevertReason`. Useful for diagnostics.
func GetRegisteredErrorSelectors() [][4]byte {
	errorRegistryMu.RLock()
	defer errorRegistryMu.RUnlock()
	out := make([][4]byte, 0, len(errorRegistry))
	for k := range errorRegistry {
		out = append(out, k)
	}
	return out
}

func init() {
	registerErrorsFromABI(afiABICached)
}

// DecodeRevertReason decodes raw revert data (4-byte selector + payload).
// Returns nil when the data is too short or doesn't match a known selector.
//
// Built-in Solidity Error(string) / Panic(uint256) are always tried in
// addition to the registered custom errors.
func DecodeRevertReason(data []byte) *DecodedRevert {
	if len(data) < 4 {
		return nil
	}
	var sel [4]byte
	copy(sel[:], data[:4])
	payload := data[4:]

	// Built-in Error(string)
	if hex.EncodeToString(sel[:]) == errorStringSelectorHex {
		args, err := errorStringDef.Inputs.Unpack(payload)
		if err == nil {
			return &DecodedRevert{Name: "Error", Signature: "Error(string)", Args: args}
		}
	}
	// Built-in Panic(uint256)
	if hex.EncodeToString(sel[:]) == panicSelectorHex {
		args, err := panicDef.Inputs.Unpack(payload)
		if err == nil {
			return &DecodedRevert{Name: "Panic", Signature: "Panic(uint256)", Args: args}
		}
	}

	errorRegistryMu.RLock()
	def, ok := errorRegistry[sel]
	errorRegistryMu.RUnlock()
	if !ok {
		return nil
	}
	args, err := def.Inputs.Unpack(payload)
	if err != nil {
		return nil
	}
	return &DecodedRevert{Name: def.Name, Signature: def.Sig, Args: args}
}

// extractRevertData pulls the raw revert hex out of a CallContract error if available.
// Returns nil when the error doesn't expose data (e.g. it's a network error, not a revert).
func extractRevertData(err error) []byte {
	if err == nil {
		return nil
	}
	// go-ethereum RPC errors implement DataError with ErrorData() returning a hex string.
	if de, ok := err.(rpc.DataError); ok {
		if d, ok := de.ErrorData().(string); ok {
			return common.FromHex(d)
		}
	}
	// As a fallback, scan the error message for "0x..." revert data.
	msg := err.Error()
	if i := strings.Index(msg, "0x"); i >= 0 {
		tail := msg[i:]
		// Pick the longest leading hex run.
		end := 2
		for end < len(tail) && isHexDigit(tail[end]) {
			end++
		}
		if end >= 10 { // at least selector (4 bytes = 8 hex + 0x prefix)
			return common.FromHex(tail[:end])
		}
	}
	return nil
}

func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
