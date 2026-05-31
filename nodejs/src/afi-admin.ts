import { encodeFunctionData } from "viem"
import { AFI_ABI, OWNABLE2STEP_ABI } from "./constants.js"
import { ZERO_ADDRESS } from "./address.js"
import type { Address, Hex } from "./types.js"

/** Mirrors Afi.MAX_FEE_BPS — kept in sync manually with the Solidity contract. */
export const MAX_FEE_BPS = 50

function assertFeeBps(bps: number, label = "feeBps"): void {
  if (!Number.isInteger(bps) || bps < 0 || bps > MAX_FEE_BPS) {
    throw new Error(`${label} must be an integer in [0, ${MAX_FEE_BPS}], got ${bps}`)
  }
}

function assertNonZeroAddress(addr: Address, label: string): void {
  if (addr.toLowerCase() === ZERO_ADDRESS) {
    throw new Error(`${label} cannot be the zero address`)
  }
}

/**
 * Owner-only. Encode `pause()` calldata for the Afi router.
 */
export function encodeAfiPause(): Hex {
  return encodeFunctionData({
    abi: AFI_ABI,
    functionName: "pause",
    args: [],
  })
}

/**
 * Owner-only. Encode `unpause()` calldata for the Afi router.
 */
export function encodeAfiUnpause(): Hex {
  return encodeFunctionData({
    abi: AFI_ABI,
    functionName: "unpause",
    args: [],
  })
}

/**
 * Owner-only. Encode `setTreasury(address)` calldata for the Afi router.
 */
export function encodeAfiSetTreasury(treasury: Address): Hex {
  assertNonZeroAddress(treasury, "treasury")
  return encodeFunctionData({
    abi: AFI_ABI,
    functionName: "setTreasury",
    args: [treasury],
  })
}

/**
 * Owner-only. Encode `setFeeBps(uint16)` calldata for the Afi router.
 */
export function encodeAfiSetFeeBps(feeBps: number): Hex {
  assertFeeBps(feeBps)
  return encodeFunctionData({
    abi: AFI_ABI,
    functionName: "setFeeBps",
    args: [feeBps],
  })
}

/**
 * Owner-only. Encode `setUserFeeBps(address, uint16)` calldata for the Afi router.
 */
export function encodeAfiSetUserFeeBps(user: Address, feeBps: number): Hex {
  assertFeeBps(feeBps)
  return encodeFunctionData({
    abi: AFI_ABI,
    functionName: "setUserFeeBps",
    args: [user, feeBps],
  })
}

/**
 * Owner-only. Encode `setUserFeeBpsBatch(address[], uint16[])` calldata
 * for the Afi router.
 */
export function encodeAfiSetUserFeeBpsBatch(users: Address[], feesBps: number[]): Hex {
  if (users.length !== feesBps.length) {
    throw new Error(`users.length (${users.length}) must equal feesBps.length (${feesBps.length})`)
  }
  for (const bps of feesBps) assertFeeBps(bps)
  return encodeFunctionData({
    abi: AFI_ABI,
    functionName: "setUserFeeBpsBatch",
    args: [users, feesBps],
  })
}

/**
 * Owner-only. Encode `clearUserFeeBps(address)` calldata for the Afi router.
 */
export function encodeAfiClearUserFeeBps(user: Address): Hex {
  return encodeFunctionData({
    abi: AFI_ABI,
    functionName: "clearUserFeeBps",
    args: [user],
  })
}

/**
 * Owner-only. Encode `resetAnyUserOverride()` calldata for the Afi router.
 */
export function encodeAfiResetAnyUserOverride(): Hex {
  return encodeFunctionData({
    abi: AFI_ABI,
    functionName: "resetAnyUserOverride",
    args: [],
  })
}

/**
 * Owner-only. Encode `addRule(address)` calldata for the Afi router.
 */
export function encodeAfiAddRule(rule: Address): Hex {
  assertNonZeroAddress(rule, "rule")
  return encodeFunctionData({
    abi: AFI_ABI,
    functionName: "addRule",
    args: [rule],
  })
}

/**
 * Owner-only. Encode `clearRules()` calldata for the Afi router.
 */
export function encodeAfiClearRules(): Hex {
  return encodeFunctionData({
    abi: AFI_ABI,
    functionName: "clearRules",
    args: [],
  })
}

/**
 * Owner-only. Encode `setOperator(address, bool)` calldata for the Afi router.
 */
export function encodeAfiSetOperator(operator: Address, value: boolean): Hex {
  return encodeFunctionData({
    abi: AFI_ABI,
    functionName: "setOperator",
    args: [operator, value],
  })
}

/**
 * Owner-only. Encode `rescueTokens(address, uint256, address)` calldata for
 * the Afi router.
 */
export function encodeAfiRescueTokens(token: Address, value: bigint, to: Address): Hex {
  return encodeFunctionData({
    abi: AFI_ABI,
    functionName: "rescueTokens",
    args: [token, value, to],
  })
}

/**
 * Encode calldata for `acceptOwnership()` — required to complete an
 * Ownable2Step transfer after the previous owner called transferOwnership(...).
 * The pending owner must be msg.sender.
 */
export function encodeAcceptOwnership(): Hex {
  return encodeFunctionData({
    abi: OWNABLE2STEP_ABI,
    functionName: "acceptOwnership",
    args: [],
  })
}
