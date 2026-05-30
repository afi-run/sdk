import { encodeFunctionData } from "viem"
import { AFI_ABI, AFI_ADDRESS, ERC20_ABI } from "./constants.js"
import type { Address, Hex, Quote } from "./types.js"

/**
 * Pre-encoded transaction payload that can be fed straight into a wallet client,
 * Wagmi / RainbowKit, MetaMask SDK, Safe SDK, Frame, or any other signer that
 * accepts `{to, data, value}`.
 */
export interface EncodedTx {
  to:    Address
  data:  Hex
  /** Always 0n for AFI swaps and ERC-20 approvals (the contracts don't accept native ETH). */
  value: bigint
}

/**
 * Builds the AFI router calldata for `quote`. The result can be signed and
 * submitted with any signer — useful when the private key is held by a wallet
 * connector (Wagmi, RainbowKit, MetaMask, hardware wallets) instead of the SDK.
 *
 * @example
 * import { encodeSwap } from "@afi-run/sdk"
 *
 * const tx = encodeSwap(quote)
 * const hash = await walletClient.sendTransaction(tx) // viem
 */
export function encodeSwap(quote: Quote): EncodedTx {
  const data = encodeFunctionData({
    abi: AFI_ABI,
    functionName: "swap",
    args: [quote.tokenIn, quote.amountInWei, quote.tokenOut, quote.minOutWei, quote.steps],
  })
  return { to: AFI_ADDRESS, data, value: 0n }
}

/**
 * Builds the ERC-20 `approve` calldata for the AFI router. Combine with
 * `encodeSwap` to issue both transactions through an external signer.
 */
export function encodeApprove(token: Address, amountWei: bigint): EncodedTx {
  const data = encodeFunctionData({
    abi: ERC20_ABI,
    functionName: "approve",
    args: [AFI_ADDRESS, amountWei],
  })
  return { to: token, data, value: 0n }
}

/**
 * Builds the ERC-20 `approve(AFI, 0)` calldata — i.e. revoke the AFI router's
 * allowance on `token`. Equivalent to `encodeApprove(token, 0n)`.
 */
export function encodeRevoke(token: Address): EncodedTx {
  return encodeApprove(token, 0n)
}
