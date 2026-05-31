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
 * Builds the AFI router calldata for `swapFor(user, tokenIn, amountIn, tokenOut, minOut, params)`.
 * Operator-only on-chain — used by the AFI backend / managed bots to execute swaps
 * on behalf of a user that already approved the router.
 */
export interface SwapForArgs {
  user: Address
  tokenIn: Address
  amountInWei: bigint
  tokenOut: Address
  minOutWei: bigint
  /** Encoded route steps (typically `quote.steps`). */
  steps: Hex
}

export function encodeSwapFor(args: SwapForArgs): EncodedTx {
  const data = encodeFunctionData({
    abi: AFI_ABI,
    functionName: "swapFor",
    args: [args.user, args.tokenIn, args.amountInWei, args.tokenOut, args.minOutWei, args.steps],
  })
  return { to: AFI_ADDRESS, data, value: 0n }
}

/**
 * Builds the AFI router calldata for `batchSwapFor(tuple[])`. Each entry mirrors
 * `swapFor` and is executed sequentially on-chain.
 */
export function encodeBatchSwapFor(swaps: readonly SwapForArgs[]): EncodedTx {
  const tuples = swaps.map((s) => ({
    user: s.user,
    tokenIn: s.tokenIn,
    amountIn: s.amountInWei,
    tokenOut: s.tokenOut,
    minOut: s.minOutWei,
    params: s.steps,
  }))
  const data = encodeFunctionData({
    abi: AFI_ABI,
    functionName: "batchSwapFor",
    args: [tuples],
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
