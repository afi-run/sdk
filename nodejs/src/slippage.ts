/**
 * Multiplies `amount` by (1 - slippagePct/100), rounding down.
 * Useful when computing your own `minOut` from a raw `amountOut`.
 *
 * `slippagePct` is in percent (0.5 = 0.5%, 1.25 = 1.25%). Negative values are
 * clamped to 0 (i.e. no slippage applied).
 */
export function applySlippage(amount: bigint, slippagePct: number): bigint {
  const bps = Math.max(0, Math.round(slippagePct * 100))
  if (bps === 0) return amount
  if (bps >= 10_000) return 0n
  return (amount * BigInt(10_000 - bps)) / 10_000n
}

/** Alias of {@link applySlippage} expressed as `minOut`. */
export function calculateMinOut(amountOutWei: bigint, slippagePct: number): bigint {
  return applySlippage(amountOutWei, slippagePct)
}
