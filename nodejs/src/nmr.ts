import { encodeFunctionData } from "viem"
import { NMR_ABI } from "./constants.js"
import type { Address, Hex } from "./types.js"

/**
 * Encode an NMR `requestOperation(asset, amount, params)` call. The result is
 * calldata only — pair with the chain's NMR address (see `NMR_ADDRESSES`) when
 * sending via a wallet client.
 */
export function encodeNMRRequestOperation(asset: Address, amount: bigint, params: Hex): Hex {
  return encodeFunctionData({
    abi: NMR_ABI,
    functionName: "requestOperation",
    args: [asset, amount, params],
  })
}

/** Encode an NMR `swap(asset, amount, minOut, params)` call. */
export function encodeNMRSwap(asset: Address, amount: bigint, minOut: bigint, params: Hex): Hex {
  return encodeFunctionData({
    abi: NMR_ABI,
    functionName: "swap",
    args: [asset, amount, minOut, params],
  })
}

/** Encode an NMR `loan(user, asset, amount, minOut, params)` call. */
export function encodeNMRLoan(
  user:   Address,
  asset:  Address,
  amount: bigint,
  minOut: bigint,
  params: Hex,
): Hex {
  return encodeFunctionData({
    abi: NMR_ABI,
    functionName: "loan",
    args: [user, asset, amount, minOut, params],
  })
}

/** Encode an NMR `sweepProfit(asset, amount)` call. */
export function encodeNMRSweepProfit(asset: Address, amount: bigint): Hex {
  return encodeFunctionData({
    abi: NMR_ABI,
    functionName: "sweepProfit",
    args: [asset, amount],
  })
}

/** Encode an NMR `setTreasury(treasury)` call. */
export function encodeNMRSetTreasury(treasury: Address): Hex {
  return encodeFunctionData({
    abi: NMR_ABI,
    functionName: "setTreasury",
    args: [treasury],
  })
}
