/** Converts a raw bigint wei amount to a human-readable decimal string. */
export function formatAmount(amount: bigint, decimals: number): string {
  const divisor = 10n ** BigInt(decimals)
  const whole = amount / divisor
  const frac = amount % divisor
  if (frac === 0n) return whole.toString()
  const fracStr = frac.toString().padStart(decimals, "0").replace(/0+$/, "")
  return `${whole}.${fracStr}`
}

/** Converts a raw bigint wei amount to a human-readable decimal string.
 *  @example formatUnits(1000_000000n, 6) // "1000" */
export function formatUnits(amount: bigint, decimals: number): string {
  return formatAmount(amount, decimals)
}

/** Converts a human-readable decimal string to raw bigint wei.
 *  @example parseUnits("1000", 6) // 1000_000000n */
export function parseUnits(amount: string, decimals: number): bigint {
  const [whole, frac = ""] = amount.split(".")
  const fracPadded = frac.padEnd(decimals, "0").slice(0, decimals)
  return BigInt(whole) * 10n ** BigInt(decimals) + BigInt(fracPadded || "0")
}
