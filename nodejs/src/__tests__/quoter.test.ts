import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { fetchQuote } from "../quoter.js"
import type { SwapParams } from "../types.js"

const USDC = "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913" as const
const WETH = "0x4200000000000000000000000000000000000006" as const

const baseParams: SwapParams = {
  tokenIn: USDC,
  tokenOut: WETH,
  amountIn: 1000_000000n,
  slippage: 0.5,
}

const successData = {
  tokenIn: USDC,
  tokenOut: WETH,
  amountInRaw: "1000000000",
  amountOutRaw: "500000000000000000",
  minOutRaw: "497500000000000000",
  steps: "0xabcdef",
  slippage: 0.5,
  path: [USDC, WETH],
}

beforeEach(() => {
  vi.stubGlobal("fetch", vi.fn())
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe("fetchQuote", () => {
  it("returns a valid Quote on success", async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify({ status: "success", data: successData }), { status: 200 }),
    )

    const quote = await fetchQuote(baseParams, 6, 35, "https://rpc.example.com")

    expect(quote.tokenIn).toBe(USDC)
    expect(quote.tokenOut).toBe(WETH)
    expect(quote.amountInWei).toBe(1000000000n)
    expect(quote.amountOutWei).toBe(500000000000000000n)
    expect(quote.minOutWei).toBe(497500000000000000n)
    expect(quote.steps).toBe("0xabcdef")
    expect(quote.slippage).toBe(0.5)
    expect(quote.feeBps).toBe(35)
    expect(quote.path).toEqual([USDC, WETH])
  })

  it("sends correct request body to quoter", async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify({ status: "success", data: successData }), { status: 200 }),
    )

    await fetchQuote(baseParams, 6, 35, "https://my-rpc.example.com")

    const [url, options] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe("https://rpc.afi.run/quoter")

    const body = JSON.parse(options.body as string)
    expect(body.network).toBe("base")
    expect(body.tokenIn).toBe(USDC)
    expect(body.tokenOut).toBe(WETH)
    expect(body.amountIn).toBe("1000")     // formatted, not raw
    expect(body.slippage).toBe(0.5)
    expect(body.rpcUrl).toBe("https://my-rpc.example.com")
    expect(body.maxHops).toBe(4)
  })

  it("correctly formats amountIn with decimals", async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify({ status: "success", data: successData }), { status: 200 }),
    )

    // 1.5 USDC = 1_500000 raw
    await fetchQuote({ ...baseParams, amountIn: 1_500000n }, 6, 35, "https://rpc.example.com")

    const [, options] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    const body = JSON.parse(options.body as string)
    expect(body.amountIn).toBe("1.5")
  })

  it("throws QuoteError on non-success status", async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(
        JSON.stringify({ status: "error", message: "no route found" }),
        { status: 200 },
      ),
    )

    await expect(fetchQuote(baseParams, 6, 35, "https://rpc.example.com")).rejects.toThrow(
      "no route found",
    )
  })

  it("throws QuoteError when minOutRaw is zero", async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(
        JSON.stringify({ status: "success", data: { ...successData, minOutRaw: "0" } }),
        { status: 200 },
      ),
    )

    await expect(fetchQuote(baseParams, 6, 35, "https://rpc.example.com")).rejects.toThrow(
      "zero minOut",
    )
  })

  it("throws QuoteError on HTTP error", async () => {
    vi.mocked(fetch).mockResolvedValue(new Response("", { status: 502 }))

    await expect(fetchQuote(baseParams, 6, 35, "https://rpc.example.com")).rejects.toThrow(
      "HTTP 502",
    )
  })

  it("throws QuoteError on network failure", async () => {
    vi.mocked(fetch).mockRejectedValue(new Error("ECONNREFUSED"))

    await expect(fetchQuote(baseParams, 6, 35, "https://rpc.example.com")).rejects.toThrow(
      "ECONNREFUSED",
    )
  })

  it("uses fallback message when error has no message field", async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify({ status: "error" }), { status: 200 }),
    )

    await expect(fetchQuote(baseParams, 6, 35, "https://rpc.example.com")).rejects.toThrow(
      "unknown error from quoter",
    )
  })
})
