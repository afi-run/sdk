package afi

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// ReferralHardCapBps is the absolute, immutable ceiling for any referral fee on
// the AfiReferralRouter (mirrors AfiReferralRouter.HARD_CAP_BPS = 0.10%).
const ReferralHardCapBps uint16 = 10

// referralRouterParsedABI is the lazily-parsed AfiReferralRouter ABI — it relies
// on the AfiReferralRouterABI constant defined in constants.go.
var referralRouterParsedABI = func() abi.ABI {
	a, err := abi.JSON(strings.NewReader(AfiReferralRouterABI))
	if err != nil {
		panic("invalid AfiReferralRouterABI: " + err.Error())
	}
	return a
}()

// ReferralRouterAddress resolves the deployed AfiReferralRouter for chainID.
// Returns the resolved Network on success, or a friendly error if the chain is
// unknown or the router is not deployed there.
func ReferralRouterAddress(chainID int64) (common.Address, Network, error) {
	return addressForChain("ReferralRouterAddress", ReferralRouterAddresses, chainID)
}

// EncodeSwapWithReferral builds calldata for
// swapWithReferral(tokenIn, amountIn, tokenOut, minOut, params, referrer, referralBps).
// Spends the caller's own funds; the output (net of fee) is returned to the caller.
// referralBps must be <= ReferralHardCapBps (and <= the router's current maxReferralBps).
// Pass referrer = zero address and/or referralBps = 0 to disable the fee.
func EncodeSwapWithReferral(
	tokenIn common.Address,
	amountIn *big.Int,
	tokenOut common.Address,
	minOut *big.Int,
	params []byte,
	referrer common.Address,
	referralBps uint16,
) ([]byte, error) {
	if amountIn == nil || amountIn.Sign() == 0 {
		return nil, fmt.Errorf("EncodeSwapWithReferral: amountIn must be > 0")
	}
	if referralBps > ReferralHardCapBps {
		return nil, fmt.Errorf("EncodeSwapWithReferral: referralBps %d exceeds hard cap %d", referralBps, ReferralHardCapBps)
	}
	return referralRouterParsedABI.Pack("swapWithReferral", tokenIn, amountIn, tokenOut, minOut, params, referrer, referralBps)
}

// EncodeSwapWithReferralFor builds calldata for
// swapWithReferralFor(user, tokenIn, amountIn, tokenOut, minOut, params, referrer, referralBps).
// Spends `user`'s funds (the caller must hold a non-expired delegate allowance
// for (user, tokenIn) covering amountIn) and the output always goes to `user`.
func EncodeSwapWithReferralFor(
	user common.Address,
	tokenIn common.Address,
	amountIn *big.Int,
	tokenOut common.Address,
	minOut *big.Int,
	params []byte,
	referrer common.Address,
	referralBps uint16,
) ([]byte, error) {
	if user == (common.Address{}) {
		return nil, fmt.Errorf("EncodeSwapWithReferralFor: user cannot be zero address")
	}
	if amountIn == nil || amountIn.Sign() == 0 {
		return nil, fmt.Errorf("EncodeSwapWithReferralFor: amountIn must be > 0")
	}
	if referralBps > ReferralHardCapBps {
		return nil, fmt.Errorf("EncodeSwapWithReferralFor: referralBps %d exceeds hard cap %d", referralBps, ReferralHardCapBps)
	}
	return referralRouterParsedABI.Pack("swapWithReferralFor", user, tokenIn, amountIn, tokenOut, minOut, params, referrer, referralBps)
}

// EncodeReferralClaim builds calldata for claim(token, to): a referrer withdraws
// their accrued fee for a single token to `to`.
func EncodeReferralClaim(token, to common.Address) ([]byte, error) {
	if to == (common.Address{}) {
		return nil, fmt.Errorf("EncodeReferralClaim: to cannot be zero address")
	}
	return referralRouterParsedABI.Pack("claim", token, to)
}

// EncodeReferralClaimMany builds calldata for claimMany(tokens, to): a referrer
// withdraws accrued fees for several tokens at once.
func EncodeReferralClaimMany(tokens []common.Address, to common.Address) ([]byte, error) {
	if to == (common.Address{}) {
		return nil, fmt.Errorf("EncodeReferralClaimMany: to cannot be zero address")
	}
	if len(tokens) == 0 {
		return nil, fmt.Errorf("EncodeReferralClaimMany: tokens cannot be empty")
	}
	return referralRouterParsedABI.Pack("claimMany", tokens, to)
}

// EncodeSetDelegateAllowance builds calldata for
// setDelegateAllowance(token, delegate, amount, deadline). Authorizes `delegate`
// to swap up to `amount` of `token` on the caller's behalf until `deadline`
// (unix seconds). This is a set (not an increase); pass amount = 0 to disable.
// `amount` must fit in uint208 and `deadline` in uint48.
func EncodeSetDelegateAllowance(token, delegate common.Address, amount *big.Int, deadline uint64) ([]byte, error) {
	if delegate == (common.Address{}) {
		return nil, fmt.Errorf("EncodeSetDelegateAllowance: delegate cannot be zero address")
	}
	if amount == nil || amount.Sign() < 0 {
		return nil, fmt.Errorf("EncodeSetDelegateAllowance: amount must be >= 0")
	}
	if amount.BitLen() > 208 {
		return nil, fmt.Errorf("EncodeSetDelegateAllowance: amount exceeds uint208")
	}
	if deadline > (1<<48)-1 {
		return nil, fmt.Errorf("EncodeSetDelegateAllowance: deadline exceeds uint48")
	}
	return referralRouterParsedABI.Pack("setDelegateAllowance", token, delegate, amount, big.NewInt(int64(deadline)))
}

// EncodeRevokeDelegate builds calldata for revokeDelegate(token, delegate): the
// caller revokes a delegate's authorization for `token` entirely.
func EncodeRevokeDelegate(token, delegate common.Address) ([]byte, error) {
	if delegate == (common.Address{}) {
		return nil, fmt.Errorf("EncodeRevokeDelegate: delegate cannot be zero address")
	}
	return referralRouterParsedABI.Pack("revokeDelegate", token, delegate)
}

// --- Owner-only controls -----------------------------------------------------

// EncodeReferralPause builds calldata for the router's pause() (owner-only).
func EncodeReferralPause() ([]byte, error) { return referralRouterParsedABI.Pack("pause") }

// EncodeReferralUnpause builds calldata for the router's unpause() (owner-only).
func EncodeReferralUnpause() ([]byte, error) { return referralRouterParsedABI.Pack("unpause") }

// EncodeReferralSetMaxReferralBps builds calldata for setMaxReferralBps(uint16)
// (owner-only). Rejects values above ReferralHardCapBps (10).
func EncodeReferralSetMaxReferralBps(bps uint16) ([]byte, error) {
	if bps > ReferralHardCapBps {
		return nil, fmt.Errorf("EncodeReferralSetMaxReferralBps: bps %d exceeds hard cap %d", bps, ReferralHardCapBps)
	}
	return referralRouterParsedABI.Pack("setMaxReferralBps", bps)
}

// EncodeReferralRescueTokens builds calldata for rescueTokens(token, to)
// (owner-only): sweeps only the balance NOT owed to referrers.
func EncodeReferralRescueTokens(token, to common.Address) ([]byte, error) {
	if to == (common.Address{}) {
		return nil, fmt.Errorf("EncodeReferralRescueTokens: to cannot be zero address")
	}
	return referralRouterParsedABI.Pack("rescueTokens", token, to)
}
