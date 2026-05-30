import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { AFI_ADDRESS, MULTICALL3_ADDRESS } from "../constants.js"
import { fetchTokenInfo, fetchTokenInfoBatch } from "../multicall.js"
import type { PublicClient, Transport } from "viem"
import type { base } from "viem/chains"

type MockPublicClient = Partial<PublicClient<Transport, typeof base>> & {
  multicall: ReturnType<typeof vi.fn>
}

const TOKEN = "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913" as const
const OWNER = "0x1234567890123456789012345678901234567890" as const

let pub: MockPublicClient

beforeEach(() => {
  pub = { multicall: vi.fn() }
})

afterEach(() => { vi.clearAllMocks() })

describe("fetchTokenInfo", () => {
  it("returns metadata only when owner is omitted", async () => {
    pub.multicall.mockResolvedValue([
      { status: "success", result: "USDC" },
      { status: "success", result: "USD Coin" },
      { status: "success", result: 6 },
    ])

    const info = await fetchTokenInfo(TOKEN, pub as any)

    expect(info).toEqual({ address: TOKEN, symbol: "USDC", name: "USD Coin", decimals: 6 })
    expect(info.balance).toBeUndefined()
    expect(info.allowance).toBeUndefined()
  })

  it("includes balance + allowance when owner is provided", async () => {
    pub.multicall.mockResolvedValue([
      { status: "success", result: "USDC" },
      { status: "success", result: "USD Coin" },
      { status: "success", result: 6 },
      { status: "success", result: 1_000_000n },
      { status: "success", result: 500_000n },
    ])

    const info = await fetchTokenInfo(TOKEN, pub as any, OWNER)

    expect(info.owner).toBe(OWNER)
    expect(info.balance).toBe(1_000_000n)
    expect(info.allowance).toBe(500_000n)
  })

  it("uses Multicall3 address and AFI as allowance spender", async () => {
    pub.multicall.mockResolvedValue([
      { status: "success", result: "X" },
      { status: "success", result: "X" },
      { status: "success", result: 18 },
      { status: "success", result: 0n },
      { status: "success", result: 0n },
    ])

    await fetchTokenInfo(TOKEN, pub as any, OWNER)

    const call = pub.multicall.mock.calls[0][0]
    expect(call.multicallAddress).toBe(MULTICALL3_ADDRESS)
    expect(call.allowFailure).toBe(true)
    const allowanceCall = call.contracts[4]
    expect(allowanceCall.functionName).toBe("allowance")
    expect(allowanceCall.args).toEqual([OWNER, AFI_ADDRESS])
  })

  it("tolerates symbol/name reverts (non-standard tokens)", async () => {
    pub.multicall.mockResolvedValue([
      { status: "failure", error: new Error("revert") },
      { status: "failure", error: new Error("revert") },
      { status: "success", result: 18 },
    ])

    const info = await fetchTokenInfo(TOKEN, pub as any)
    expect(info.symbol).toBe("")
    expect(info.name).toBe("")
    expect(info.decimals).toBe(18)
  })

  it("throws when decimals reverts", async () => {
    pub.multicall.mockResolvedValue([
      { status: "success", result: "X" },
      { status: "success", result: "X" },
      { status: "failure", error: new Error("revert") },
    ])

    await expect(fetchTokenInfo(TOKEN, pub as any)).rejects.toThrow(/decimals/)
  })

  it("treats the zero address as no-owner", async () => {
    pub.multicall.mockResolvedValue([
      { status: "success", result: "USDC" },
      { status: "success", result: "USD Coin" },
      { status: "success", result: 6 },
    ])

    const info = await fetchTokenInfo(TOKEN, pub as any, "0x0000000000000000000000000000000000000000")
    expect(info.balance).toBeUndefined()
    expect(info.allowance).toBeUndefined()
  })
})

describe("fetchTokenInfoBatch", () => {
  const TOKEN_A = "0xaaaa589fcd6edb6e08f4c7c32d4f71b54bda02913" as const
  const TOKEN_B = "0xbbbb589fcd6edb6e08f4c7c32d4f71b54bda02913" as const

  it("returns [] for empty input without an RPC call", async () => {
    const out = await fetchTokenInfoBatch([], pub as any)
    expect(out).toEqual([])
    expect(pub.multicall).not.toHaveBeenCalled()
  })

  it("returns one TokenInfo per input token, in order", async () => {
    pub.multicall.mockResolvedValue([
      { status: "success", result: "A" },
      { status: "success", result: "Alpha" },
      { status: "success", result: 6 },
      { status: "success", result: "B" },
      { status: "success", result: "Beta" },
      { status: "success", result: 18 },
    ])

    const out = await fetchTokenInfoBatch([TOKEN_A, TOKEN_B], pub as any)

    expect(out).toHaveLength(2)
    expect(out[0]).toEqual({ address: TOKEN_A, symbol: "A", name: "Alpha", decimals: 6 })
    expect(out[1]).toEqual({ address: TOKEN_B, symbol: "B", name: "Beta", decimals: 18 })
  })

  it("throws when balance or allowance reverts for any token", async () => {
    pub.multicall.mockResolvedValue([
      { status: "success", result: "A" },
      { status: "success", result: "Alpha" },
      { status: "success", result: 6 },
      { status: "failure", error: new Error("reverted") }, // balanceOf
      { status: "success", result: 0n },
    ])
    await expect(fetchTokenInfoBatch([TOKEN_A], pub as any, OWNER)).rejects.toThrow(/balanceOf/)

    pub.multicall.mockResolvedValue([
      { status: "success", result: "A" },
      { status: "success", result: "Alpha" },
      { status: "success", result: 6 },
      { status: "success", result: 0n },
      { status: "failure", error: new Error("reverted") }, // allowance
    ])
    await expect(fetchTokenInfoBatch([TOKEN_A], pub as any, OWNER)).rejects.toThrow(/allowance/)
  })

  it("packs balance + allowance per token when owner is provided", async () => {
    pub.multicall.mockResolvedValue([
      // token A: symbol, name, decimals, balance, allowance
      { status: "success", result: "A" },
      { status: "success", result: "Alpha" },
      { status: "success", result: 6 },
      { status: "success", result: 100n },
      { status: "success", result: 50n },
      // token B
      { status: "success", result: "B" },
      { status: "success", result: "Beta" },
      { status: "success", result: 18 },
      { status: "success", result: 200n },
      { status: "success", result: 150n },
    ])

    const out = await fetchTokenInfoBatch([TOKEN_A, TOKEN_B], pub as any, OWNER)

    expect(out[0].balance).toBe(100n)
    expect(out[0].allowance).toBe(50n)
    expect(out[1].balance).toBe(200n)
    expect(out[1].allowance).toBe(150n)

    const calls = pub.multicall.mock.calls[0][0].contracts
    expect(calls).toHaveLength(10) // 2 tokens × 5 calls
  })
})
