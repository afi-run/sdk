import { describe, expect, it } from "vitest"
import { decodeFunctionData, getAddress } from "viem"
import { AFI_ABI, AFI_ADDRESS, ERC20_ABI } from "../constants.js"
import {
  encodeApprove,
  encodeBatchSwapFor,
  encodeRevoke,
  encodeSwap,
  encodeSwapFor,
} from "../encode.js"
import type { SwapForArgs } from "../encode.js"
import type { Quote } from "../types.js"

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
  path:         [],
  hops:         [],
  slippage:     0.5,
  feeBps:       35,
  tokenInPrice:  "1",
  tokenOutPrice: "1",
  createdAt:     Date.now(),
  network:       "base",
  maxHops:       2,
}

const TOKEN = "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913" as const

describe("encodeSwap", () => {
  it("targets the AFI router with value=0", () => {
    const tx = encodeSwap(sampleQuote)
    expect(tx.to).toBe(AFI_ADDRESS)
    expect(tx.value).toBe(0n)
    expect(tx.data).toMatch(/^0x[0-9a-f]+$/i)
  })

  it("encodes the swap(...) call with the quote fields", () => {
    const tx = encodeSwap(sampleQuote)
    const decoded = decodeFunctionData({ abi: AFI_ABI, data: tx.data })
    expect(decoded.functionName).toBe("swap")
    const args = decoded.args as readonly unknown[]
    expect(getAddress(args[0] as string)).toBe(getAddress(sampleQuote.tokenIn))
    expect(args[1]).toBe(sampleQuote.amountInWei)
    expect(getAddress(args[2] as string)).toBe(getAddress(sampleQuote.tokenOut))
    expect(args[3]).toBe(sampleQuote.minOutWei)
    expect(args[4]).toBe(sampleQuote.steps)
  })
})

describe("encodeApprove", () => {
  it("targets the token with approve(AFI, amount) calldata", () => {
    const tx = encodeApprove(TOKEN, 1_000_000n)
    expect(tx.to).toBe(TOKEN)
    expect(tx.value).toBe(0n)
    const decoded = decodeFunctionData({ abi: ERC20_ABI, data: tx.data })
    expect(decoded.functionName).toBe("approve")
    const args = decoded.args as readonly unknown[]
    expect(getAddress(args[0] as string)).toBe(getAddress(AFI_ADDRESS))
    expect(args[1]).toBe(1_000_000n)
  })
})

describe("encodeRevoke", () => {
  it("equals encodeApprove(token, 0n)", () => {
    expect(encodeRevoke(TOKEN)).toEqual(encodeApprove(TOKEN, 0n))
  })
})

const sampleSwapFor: SwapForArgs = {
  user:        "0x1111111111111111111111111111111111111111",
  tokenIn:     "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
  amountInWei: 1_000_000_000n,
  tokenOut:    "0x4200000000000000000000000000000000000006",
  minOutWei:   495_000_000_000_000_000n,
  steps:       "0xdeadbeef",
}

describe("encodeSwapFor", () => {
  it("targets the AFI router with value=0", () => {
    const tx = encodeSwapFor(sampleSwapFor)
    expect(tx.to).toBe(AFI_ADDRESS)
    expect(tx.value).toBe(0n)
    expect(tx.data).toMatch(/^0x[0-9a-f]+$/i)
  })

  it("encodes swapFor(...) with the args in order", () => {
    const tx = encodeSwapFor(sampleSwapFor)
    const decoded = decodeFunctionData({ abi: AFI_ABI, data: tx.data })
    expect(decoded.functionName).toBe("swapFor")
    const args = decoded.args as readonly unknown[]
    expect(getAddress(args[0] as string)).toBe(getAddress(sampleSwapFor.user))
    expect(getAddress(args[1] as string)).toBe(getAddress(sampleSwapFor.tokenIn))
    expect(args[2]).toBe(sampleSwapFor.amountInWei)
    expect(getAddress(args[3] as string)).toBe(getAddress(sampleSwapFor.tokenOut))
    expect(args[4]).toBe(sampleSwapFor.minOutWei)
    expect(args[5]).toBe(sampleSwapFor.steps)
  })
})

describe("encodeBatchSwapFor", () => {
  it("targets the AFI router with value=0", () => {
    const tx = encodeBatchSwapFor([sampleSwapFor])
    expect(tx.to).toBe(AFI_ADDRESS)
    expect(tx.value).toBe(0n)
    expect(tx.data).toMatch(/^0x[0-9a-f]+$/i)
  })

  it("encodes batchSwapFor(tuple[]) mapping every entry", () => {
    const second: SwapForArgs = { ...sampleSwapFor, user: "0x2222222222222222222222222222222222222222" }
    const tx = encodeBatchSwapFor([sampleSwapFor, second])
    const decoded = decodeFunctionData({ abi: AFI_ABI, data: tx.data })
    expect(decoded.functionName).toBe("batchSwapFor")
    const tuples = (decoded.args as readonly unknown[])[0] as readonly Record<string, unknown>[]
    expect(tuples).toHaveLength(2)
    expect(getAddress(tuples[0].user as string)).toBe(getAddress(sampleSwapFor.user))
    expect(tuples[0].amountIn).toBe(sampleSwapFor.amountInWei)
    expect(tuples[0].minOut).toBe(sampleSwapFor.minOutWei)
    expect(tuples[0].params).toBe(sampleSwapFor.steps)
    expect(getAddress(tuples[1].user as string)).toBe(getAddress(second.user))
  })

  it("encodes an empty batch", () => {
    const tx = encodeBatchSwapFor([])
    const decoded = decodeFunctionData({ abi: AFI_ABI, data: tx.data })
    expect(decoded.functionName).toBe("batchSwapFor")
    expect((decoded.args as readonly unknown[])[0]).toHaveLength(0)
  })
})
