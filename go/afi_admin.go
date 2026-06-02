package afi

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// MaxFeeBps is the maximum protocol fee (mirrors Afi.MAX_FEE_BPS).
const MaxFeeBps uint16 = 50

// afiParsedABI is the lazily-parsed Afi contract ABI — it relies on the
// AfiABI constant defined in constants.go.
var afiParsedABI = func() abi.ABI {
	a, err := abi.JSON(strings.NewReader(AfiABI))
	if err != nil {
		panic("invalid AfiABI: " + err.Error())
	}
	return a
}()

// Ownable2StepABI is a minimal ABI exposing the three Ownable2Step entry points
// used by the SDK (acceptOwnership / pendingOwner / owner). Exported so callers
// can reuse it for raw contract calls.
const Ownable2StepABI = `[
  {"type":"function","name":"acceptOwnership","inputs":[],"outputs":[],"stateMutability":"nonpayable"},
  {"type":"function","name":"pendingOwner","inputs":[],"outputs":[{"name":"","type":"address"}],"stateMutability":"view"},
  {"type":"function","name":"owner","inputs":[],"outputs":[{"name":"","type":"address"}],"stateMutability":"view"}
]`

// ownable2StepABI is the parsed Ownable2Step ABI used by EncodeAcceptOwnership
// and AcceptOwnership.
var ownable2StepABI = func() abi.ABI {
	a, err := abi.JSON(strings.NewReader(Ownable2StepABI))
	if err != nil {
		panic("invalid Ownable2StepABI: " + err.Error())
	}
	return a
}()

// EncodeAcceptOwnership builds calldata for Ownable2Step.acceptOwnership().
// The transaction sender must be the pending owner — the on-chain contract
// reverts otherwise. The 4-byte selector is 0x79ba5097.
func EncodeAcceptOwnership() ([]byte, error) {
	return ownable2StepABI.Pack("acceptOwnership")
}

// EncodeAfiPause builds calldata for Afi.pause() (owner-only).
func EncodeAfiPause() ([]byte, error) { return afiParsedABI.Pack("pause") }

// EncodeAfiUnpause builds calldata for Afi.unpause() (owner-only).
func EncodeAfiUnpause() ([]byte, error) { return afiParsedABI.Pack("unpause") }

// EncodeAfiSetTreasury builds calldata for Afi.setTreasury(address) (owner-only).
// Rejects the zero address — the on-chain contract reverts otherwise.
func EncodeAfiSetTreasury(treasury common.Address) ([]byte, error) {
	if treasury == (common.Address{}) {
		return nil, fmt.Errorf("EncodeAfiSetTreasury: treasury cannot be zero address")
	}
	return afiParsedABI.Pack("setTreasury", treasury)
}

// EncodeAfiSetFeeBps builds calldata for Afi.setFeeBps(uint16) (owner-only).
// Rejects values above MaxFeeBps (50) — the on-chain contract reverts otherwise.
func EncodeAfiSetFeeBps(feeBps uint16) ([]byte, error) {
	if feeBps > MaxFeeBps {
		return nil, fmt.Errorf("EncodeAfiSetFeeBps: feeBps %d exceeds max %d", feeBps, MaxFeeBps)
	}
	return afiParsedABI.Pack("setFeeBps", feeBps)
}

// EncodeAfiSetUserFeeBps builds calldata for Afi.setUserFeeBps(address, uint16) (owner-only).
// Rejects values above MaxFeeBps (50).
func EncodeAfiSetUserFeeBps(user common.Address, feeBps uint16) ([]byte, error) {
	if feeBps > MaxFeeBps {
		return nil, fmt.Errorf("EncodeAfiSetUserFeeBps: feeBps %d exceeds max %d", feeBps, MaxFeeBps)
	}
	return afiParsedABI.Pack("setUserFeeBps", user, feeBps)
}

// EncodeAfiSetUserFeeBpsBatch builds calldata for Afi.setUserFeeBpsBatch(address[], uint16[]) (owner-only).
// Requires len(users) == len(feesBps) and every fee <= MaxFeeBps.
func EncodeAfiSetUserFeeBpsBatch(users []common.Address, feesBps []uint16) ([]byte, error) {
	if len(users) != len(feesBps) {
		return nil, fmt.Errorf("EncodeAfiSetUserFeeBpsBatch: users (%d) and feesBps (%d) length mismatch", len(users), len(feesBps))
	}
	for i, f := range feesBps {
		if f > MaxFeeBps {
			return nil, fmt.Errorf("EncodeAfiSetUserFeeBpsBatch: feesBps[%d]=%d exceeds max %d", i, f, MaxFeeBps)
		}
	}
	return afiParsedABI.Pack("setUserFeeBpsBatch", users, feesBps)
}

// EncodeAfiClearUserFeeBps builds calldata for Afi.clearUserFeeBps(address) (owner-only).
func EncodeAfiClearUserFeeBps(user common.Address) ([]byte, error) {
	return afiParsedABI.Pack("clearUserFeeBps", user)
}

// EncodeAfiResetAnyUserOverride builds calldata for Afi.resetAnyUserOverride() (owner-only).
func EncodeAfiResetAnyUserOverride() ([]byte, error) {
	return afiParsedABI.Pack("resetAnyUserOverride")
}

// EncodeAfiAddRule builds calldata for Afi.addRule(address) (owner-only).
// Rejects the zero address.
func EncodeAfiAddRule(rule common.Address) ([]byte, error) {
	if rule == (common.Address{}) {
		return nil, fmt.Errorf("EncodeAfiAddRule: rule cannot be zero address")
	}
	return afiParsedABI.Pack("addRule", rule)
}

// EncodeAfiClearRules builds calldata for Afi.clearRules() (owner-only).
func EncodeAfiClearRules() ([]byte, error) {
	return afiParsedABI.Pack("clearRules")
}

// EncodeAfiSetOperator builds calldata for Afi.setOperator(address, bool) (owner-only).
func EncodeAfiSetOperator(operator common.Address, value bool) ([]byte, error) {
	return afiParsedABI.Pack("setOperator", operator, value)
}

// EncodeAfiRescueTokens builds calldata for Afi.rescueTokens(address, uint256, address) (owner-only).
func EncodeAfiRescueTokens(token common.Address, value *big.Int, to common.Address) ([]byte, error) {
	return afiParsedABI.Pack("rescueTokens", token, value, to)
}
