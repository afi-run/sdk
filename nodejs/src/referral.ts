import { encodeFunctionData } from "viem"
import { AFI_REFERRAL_ROUTER_ABI, REFERRAL_ROUTER_ADDRESSES } from "./constants.js"
import { ZERO_ADDRESS } from "./address.js"
import type { Address, Hex } from "./types.js"

/**
 * Absolute, immutable ceiling for any referral fee on the AfiReferralRouter
 * (mirrors AfiReferralRouter.HARD_CAP_BPS = 0.10%).
 */
export const REFERRAL_HARD_CAP_BPS = 10

const UINT208_MAX = (1n << 208n) - 1n
const UINT48_MAX = 2 ** 48 - 1

function assertNonZeroAddress(addr: Address, label: string): void {
  if (addr.toLowerCase() === ZERO_ADDRESS) {
    throw new Error(`${label} cannot be the zero address`)
  }
}

function assertReferralBps(bps: number): void {
  if (!Number.isInteger(bps) || bps < 0 || bps > REFERRAL_HARD_CAP_BPS) {
    throw new Error(`referralBps must be an integer in [0, ${REFERRAL_HARD_CAP_BPS}], got ${bps}`)
  }
}

/**
 * Resolve the deployed AfiReferralRouter address for a chain ID.
 * Throws if the router is not deployed on that chain.
 */
export function referralRouterAddress(chainId: number): Address {
  const addr = REFERRAL_ROUTER_ADDRESSES[chainId]
  if (!addr) {
    throw new Error(`AfiReferralRouter not deployed on chain id ${chainId}`)
  }
  return addr
}

/**
 * Encode `swapWithReferral(tokenIn, amountIn, tokenOut, minOut, params, referrer, referralBps)`.
 * Spends the caller's own funds; the output (net of fee) is returned to the caller.
 * `referralBps` must be <= REFERRAL_HARD_CAP_BPS (and <= the router's current
 * maxReferralBps). Pass `referrer = zero address` and/or `referralBps = 0` to
 * disable the fee.
 */
export function encodeSwapWithReferral(args: {
  tokenIn: Address
  amountIn: bigint
  tokenOut: Address
  minOut: bigint
  params: Hex
  referrer: Address
  referralBps: number
}): Hex {
  if (args.amountIn <= 0n) throw new Error("amountIn must be > 0")
  assertReferralBps(args.referralBps)
  return encodeFunctionData({
    abi: AFI_REFERRAL_ROUTER_ABI,
    functionName: "swapWithReferral",
    args: [
      args.tokenIn,
      args.amountIn,
      args.tokenOut,
      args.minOut,
      args.params,
      args.referrer,
      args.referralBps,
    ],
  })
}

/**
 * Encode `swapWithReferralFor(user, tokenIn, amountIn, tokenOut, minOut, params, referrer, referralBps)`.
 * Spends `user`'s funds (the caller must hold a non-expired delegate allowance
 * for `(user, tokenIn)` covering `amountIn`); the output always goes to `user`.
 */
export function encodeSwapWithReferralFor(args: {
  user: Address
  tokenIn: Address
  amountIn: bigint
  tokenOut: Address
  minOut: bigint
  params: Hex
  referrer: Address
  referralBps: number
}): Hex {
  assertNonZeroAddress(args.user, "user")
  if (args.amountIn <= 0n) throw new Error("amountIn must be > 0")
  assertReferralBps(args.referralBps)
  return encodeFunctionData({
    abi: AFI_REFERRAL_ROUTER_ABI,
    functionName: "swapWithReferralFor",
    args: [
      args.user,
      args.tokenIn,
      args.amountIn,
      args.tokenOut,
      args.minOut,
      args.params,
      args.referrer,
      args.referralBps,
    ],
  })
}

/**
 * Encode `claim(token, to)`: a referrer withdraws their accrued fee for a single
 * token to `to`.
 */
export function encodeReferralClaim(token: Address, to: Address): Hex {
  assertNonZeroAddress(to, "to")
  return encodeFunctionData({
    abi: AFI_REFERRAL_ROUTER_ABI,
    functionName: "claim",
    args: [token, to],
  })
}

/**
 * Encode `claimMany(tokens, to)`: a referrer withdraws accrued fees for several
 * tokens at once.
 */
export function encodeReferralClaimMany(tokens: Address[], to: Address): Hex {
  assertNonZeroAddress(to, "to")
  if (tokens.length === 0) throw new Error("tokens cannot be empty")
  return encodeFunctionData({
    abi: AFI_REFERRAL_ROUTER_ABI,
    functionName: "claimMany",
    args: [tokens, to],
  })
}

/**
 * Encode `setDelegateAllowance(token, delegate, amount, deadline)`. Authorizes
 * `delegate` to swap up to `amount` of `token` on the caller's behalf until
 * `deadline` (unix seconds). This is a set (not an increase); pass `amount = 0n`
 * to disable. `amount` must fit in uint208 and `deadline` (unix seconds) in uint48.
 */
export function encodeSetDelegateAllowance(
  token: Address,
  delegate: Address,
  amount: bigint,
  deadline: number,
): Hex {
  assertNonZeroAddress(delegate, "delegate")
  if (amount < 0n || amount > UINT208_MAX) throw new Error("amount must be in [0, uint208]")
  if (!Number.isInteger(deadline) || deadline < 0 || deadline > UINT48_MAX) {
    throw new Error("deadline must be an integer in [0, uint48]")
  }
  return encodeFunctionData({
    abi: AFI_REFERRAL_ROUTER_ABI,
    functionName: "setDelegateAllowance",
    args: [token, delegate, amount, deadline],
  })
}

/**
 * Encode `revokeDelegate(token, delegate)`: the caller revokes a delegate's
 * authorization for `token` entirely.
 */
export function encodeRevokeDelegate(token: Address, delegate: Address): Hex {
  assertNonZeroAddress(delegate, "delegate")
  return encodeFunctionData({
    abi: AFI_REFERRAL_ROUTER_ABI,
    functionName: "revokeDelegate",
    args: [token, delegate],
  })
}

// ─── Owner-only controls ──────────────────────────────────────────────────────

/** Owner-only. Encode the router's `pause()`. */
export function encodeReferralPause(): Hex {
  return encodeFunctionData({ abi: AFI_REFERRAL_ROUTER_ABI, functionName: "pause", args: [] })
}

/** Owner-only. Encode the router's `unpause()`. */
export function encodeReferralUnpause(): Hex {
  return encodeFunctionData({ abi: AFI_REFERRAL_ROUTER_ABI, functionName: "unpause", args: [] })
}

/**
 * Owner-only. Encode `setMaxReferralBps(uint16)`. Rejects values above
 * REFERRAL_HARD_CAP_BPS (10).
 */
export function encodeReferralSetMaxReferralBps(bps: number): Hex {
  assertReferralBps(bps)
  return encodeFunctionData({
    abi: AFI_REFERRAL_ROUTER_ABI,
    functionName: "setMaxReferralBps",
    args: [bps],
  })
}

/**
 * Owner-only. Encode `rescueTokens(token, to)`: sweeps only the balance NOT owed
 * to referrers.
 */
export function encodeReferralRescueTokens(token: Address, to: Address): Hex {
  assertNonZeroAddress(to, "to")
  return encodeFunctionData({
    abi: AFI_REFERRAL_ROUTER_ABI,
    functionName: "rescueTokens",
    args: [token, to],
  })
}
