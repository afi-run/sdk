import { describe, expect, it } from "vitest"
import {
  bigintReplacer,
  quoteFromJSON,
  quoteToJSON,
  swapResultFromJSON,
  swapResultToJSON,
  tokenInfoFromJSON,
  tokenInfoToJSON,
} from "../serialize.js"
import type { Quote, SwapResult, TokenInfo } from "../types.js"

const sampleQuote: Quote = {
  tokenIn:      "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
  tokenOut:     "0x4200000000000000000000000000000000000006",
  amountIn:     "1000",
  amountOut:    "0.5",
  minOut:       "0.495",
  amountInWei:  1_000_000_000n,
  amountOutWei: 500_000_000_000_000_000n,
  minOutWei:    495_000_000_000_000_000n,
  steps:        "0xdeadbeef",
  path:         ["0x833589fcd6edb6e08f4c7c32d4f71b54bda02913", "0x4200000000000000000000000000000000000006"],
  hops: [{
    tokenIn:      "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
    tokenOut:     "0x4200000000000000000000000000000000000006",
    amountIn:     "1000",
    amountOut:    "0.5",
    minOut:       "0.495",
    amountInWei:  1_000_000_000n,
    amountOutWei: 500_000_000_000_000_000n,
    minOutWei:    495_000_000_000_000_000n,
    tokenInPrice:  "1",
    tokenOutPrice: "1",
    slippage:     0.5,
    type:         "v3",
    kind:         "uniswap",
    routeId:      1,
    weight:       1.0,
  }],
  slippage:      0.5,
  feeBps:        35,
  tokenInPrice:  "1",
  tokenOutPrice: "1",
  createdAt:     1_700_000_000_000,
  network:       "base",
  maxHops:       2,
}

describe("bigintReplacer", () => {
  it("converts bigints to strings inside JSON.stringify", () => {
    const out = JSON.stringify({ a: 100n, b: "x", c: 2 }, bigintReplacer)
    expect(out).toBe('{"a":"100","b":"x","c":2}')
  })
})

describe("Quote serialize roundtrip", () => {
  it("preserves all bigints across JSON", () => {
    const json = quoteToJSON(sampleQuote)
    // Ensure no bigints remain — JSON.stringify will throw if any do
    const str = JSON.stringify(json)
    expect(typeof json.amountInWei).toBe("string")
    expect(json.amountInWei).toBe("1000000000")

    const parsed = JSON.parse(str)
    const restored = quoteFromJSON(parsed)
    expect(restored.amountInWei).toBe(sampleQuote.amountInWei)
    expect(restored.minOutWei).toBe(sampleQuote.minOutWei)
    expect(restored.hops[0].amountOutWei).toBe(sampleQuote.hops[0].amountOutWei)
    expect(restored.createdAt).toBe(sampleQuote.createdAt)
  })

  it("accepts a JSON string directly", () => {
    const json = JSON.stringify(quoteToJSON(sampleQuote))
    const restored = quoteFromJSON(json)
    expect(restored.amountInWei).toBe(sampleQuote.amountInWei)
  })
})

describe("SwapResult serialize roundtrip", () => {
  const sample: SwapResult = {
    txHash:      "0xabc",
    blockNumber: 1234567890123n,
    amountIn:    1_000_000_000n,
    amountOut:   500_000_000_000_000_000n,
    tokenIn:     "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
    tokenOut:    "0x4200000000000000000000000000000000000006",
    gasUsed:     150_000n,
    effectiveGasPrice: 1_500_000_000n,
    feeWei:      150_000n * 1_500_000_000n,
    feeEth:      "0.000225",
  }
  it("preserves all bigints", () => {
    const json = swapResultToJSON(sample)
    expect(json.blockNumber).toBe("1234567890123")
    const restored = swapResultFromJSON(JSON.stringify(json))
    expect(restored.amountIn).toBe(sample.amountIn)
    expect(restored.amountOut).toBe(sample.amountOut)
    expect(restored.blockNumber).toBe(sample.blockNumber)
    expect(restored.gasUsed).toBe(sample.gasUsed)
  })
})

describe("Quote serialize edge cases", () => {
  it("fromJSON tolerates missing path/hops", () => {
    const minimal = {
      ...quoteToJSON(sampleQuote),
      path: undefined,
      hops: undefined,
    } as any
    const r = quoteFromJSON(minimal)
    expect(r.path).toEqual([])
    expect(r.hops).toEqual([])
  })
})

describe("SwapResult serialize edge cases", () => {
  it("fromJSON tolerates missing fee fields", () => {
    const json = {
      txHash: "0xabc", blockNumber: "1", amountIn: "10", amountOut: "20",
      tokenIn: "0xaa", tokenOut: "0xbb", gasUsed: "100",
      // effectiveGasPrice + feeWei missing
    } as any
    const r = swapResultFromJSON(json)
    expect(r.effectiveGasPrice).toBe(0n)
    expect(r.feeWei).toBe(0n)
  })
})

describe("TokenInfo serialize roundtrip", () => {
  it("handles tokens with balance + allowance", () => {
    const info: TokenInfo = {
      address: "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
      symbol:  "USDC",
      name:    "USD Coin",
      decimals: 6,
      owner:   "0x1234567890123456789012345678901234567890",
      balance:   1_000_000n,
      allowance: 500_000n,
    }
    const json = tokenInfoToJSON(info)
    expect(json.balance).toBe("1000000")
    const restored = tokenInfoFromJSON(JSON.stringify(json))
    expect(restored.balance).toBe(info.balance)
    expect(restored.allowance).toBe(info.allowance)
  })

  it("handles metadata-only tokens", () => {
    const info: TokenInfo = { address: "0xa", symbol: "X", name: "X", decimals: 18 }
    const restored = tokenInfoFromJSON(tokenInfoToJSON(info))
    expect(restored.balance).toBeUndefined()
    expect(restored.allowance).toBeUndefined()
  })
})
