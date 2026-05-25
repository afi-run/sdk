import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { fetchTokens } from "../info.js"

const mockBaseTokens = [
  { address: "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913", symbol: "USDC", decimals: 6, active: true },
  { address: "0x4200000000000000000000000000000000000006", symbol: "WETH", decimals: 18, active: true },
  { address: "0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf", symbol: "cbBTC", decimals: 8, active: true },
]

const successResponse = {
  status: "success",
  data: { base: mockBaseTokens },
  time: "0ms",
}

beforeEach(() => {
  vi.stubGlobal("fetch", vi.fn())
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe("fetchTokens", () => {
  it("returns base tokens on success", async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify(successResponse), { status: 200 }),
    )

    const tokens = await fetchTokens()

    expect(tokens).toHaveLength(3)
    expect(tokens[0].symbol).toBe("USDC")
    expect(tokens[0].decimals).toBe(6)
    expect(tokens[0].active).toBe(true)
    expect(tokens[1].symbol).toBe("WETH")
    expect(tokens[1].decimals).toBe(18)
  })

  it("returns address as lowercase hex string", async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify(successResponse), { status: 200 }),
    )

    const tokens = await fetchTokens()
    expect(tokens[0].address).toBe("0x833589fcd6edb6e08f4c7c32d4f71b54bda02913")
  })

  it("throws on HTTP error", async () => {
    vi.mocked(fetch).mockResolvedValue(new Response("", { status: 500 }))

    await expect(fetchTokens()).rejects.toThrow("HTTP 500")
  })

  it("throws on network failure", async () => {
    vi.mocked(fetch).mockRejectedValue(new Error("connection refused"))

    await expect(fetchTokens()).rejects.toThrow("connection refused")
  })

  it("throws when status is not success", async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify({ status: "error" }), { status: 200 }),
    )

    await expect(fetchTokens()).rejects.toThrow("no data for base network")
  })

  it("throws when base key is missing", async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify({ status: "success", data: { arbitrum: [] } }), { status: 200 }),
    )

    await expect(fetchTokens()).rejects.toThrow("no data for base network")
  })
})
