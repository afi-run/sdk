import type { Address, Hex, Hop, Quote, SwapResult, TokenInfo } from "./types.js"

/**
 * `JSON.stringify` replacer that turns `bigint` values into base-10 strings.
 *
 *     JSON.stringify(obj, bigintReplacer)
 */
export function bigintReplacer(_key: string, value: unknown): unknown {
  return typeof value === "bigint" ? value.toString() : value
}

// ─── Quote ────────────────────────────────────────────────────────────────────

export interface SerializedHop extends Omit<Hop, "amountInWei" | "amountOutWei" | "minOutWei"> {
  amountInWei: string
  amountOutWei: string
  minOutWei: string
}

export interface SerializedQuote extends Omit<Quote, "amountInWei" | "amountOutWei" | "minOutWei" | "hops"> {
  amountInWei: string
  amountOutWei: string
  minOutWei: string
  hops: SerializedHop[]
  // network / maxHops / priceBase / dexs are JSON-safe primitives — inherited via Omit
}

function hopToJSON(h: Hop): SerializedHop {
  return { ...h, amountInWei: h.amountInWei.toString(), amountOutWei: h.amountOutWei.toString(), minOutWei: h.minOutWei.toString() }
}

function hopFromJSON(h: SerializedHop): Hop {
  return { ...h, amountInWei: BigInt(h.amountInWei), amountOutWei: BigInt(h.amountOutWei), minOutWei: BigInt(h.minOutWei) }
}

/** Converts a `Quote` to a JSON-safe object (all bigints become base-10 strings). */
export function quoteToJSON(q: Quote): SerializedQuote {
  return {
    ...q,
    amountInWei:  q.amountInWei.toString(),
    amountOutWei: q.amountOutWei.toString(),
    minOutWei:    q.minOutWei.toString(),
    hops:         q.hops.map(hopToJSON),
  }
}

/** Reverses {@link quoteToJSON}. Accepts either the typed shape or a generic object/string. */
export function quoteFromJSON(j: SerializedQuote | string): Quote {
  const o = typeof j === "string" ? JSON.parse(j) as SerializedQuote : j
  return {
    ...o,
    tokenIn:      o.tokenIn as Address,
    tokenOut:     o.tokenOut as Address,
    amountInWei:  BigInt(o.amountInWei),
    amountOutWei: BigInt(o.amountOutWei),
    minOutWei:    BigInt(o.minOutWei),
    steps:        o.steps as Hex,
    path:         (o.path ?? []) as Address[],
    hops:         (o.hops ?? []).map(hopFromJSON),
  }
}

// ─── SwapResult ───────────────────────────────────────────────────────────────

export interface SerializedSwapResult extends Omit<SwapResult, "amountIn" | "amountOut" | "blockNumber" | "gasUsed" | "effectiveGasPrice" | "feeWei"> {
  amountIn: string
  amountOut: string
  blockNumber: string
  gasUsed: string
  effectiveGasPrice: string
  feeWei: string
}

export function swapResultToJSON(r: SwapResult): SerializedSwapResult {
  return {
    ...r,
    amountIn:    r.amountIn.toString(),
    amountOut:   r.amountOut.toString(),
    blockNumber: r.blockNumber.toString(),
    gasUsed:     r.gasUsed.toString(),
    effectiveGasPrice: r.effectiveGasPrice.toString(),
    feeWei:      r.feeWei.toString(),
  }
}

export function swapResultFromJSON(j: SerializedSwapResult | string): SwapResult {
  const o = typeof j === "string" ? JSON.parse(j) as SerializedSwapResult : j
  return {
    ...o,
    txHash:      o.txHash as Hex,
    tokenIn:     o.tokenIn as Address,
    tokenOut:    o.tokenOut as Address,
    amountIn:    BigInt(o.amountIn),
    amountOut:   BigInt(o.amountOut),
    blockNumber: BigInt(o.blockNumber),
    gasUsed:     BigInt(o.gasUsed),
    effectiveGasPrice: BigInt(o.effectiveGasPrice ?? "0"),
    feeWei:      BigInt(o.feeWei ?? "0"),
  }
}

// ─── TokenInfo ───────────────────────────────────────────────────────────────

export interface SerializedTokenInfo extends Omit<TokenInfo, "balance" | "allowance"> {
  balance?: string
  allowance?: string
}

export function tokenInfoToJSON(t: TokenInfo): SerializedTokenInfo {
  return {
    ...t,
    balance:   t.balance   !== undefined ? t.balance.toString()   : undefined,
    allowance: t.allowance !== undefined ? t.allowance.toString() : undefined,
  }
}

export function tokenInfoFromJSON(j: SerializedTokenInfo | string): TokenInfo {
  const o = typeof j === "string" ? JSON.parse(j) as SerializedTokenInfo : j
  return {
    ...o,
    address:   o.address as Address,
    owner:     o.owner as Address | undefined,
    balance:   o.balance   !== undefined ? BigInt(o.balance)   : undefined,
    allowance: o.allowance !== undefined ? BigInt(o.allowance) : undefined,
  }
}
