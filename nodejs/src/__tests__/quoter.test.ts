import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { fetchQuote, type QuoteRequest } from "../quoter.js"
import { NETWORK, DEX } from "../types.js"

const USDC = "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913" as const
const WETH = "0x4200000000000000000000000000000000000006" as const
const QUOTER_URL = "https://rpc.afi.run/quoter"

const baseParams: QuoteRequest = {
  tokenIn:  USDC,
  tokenOut: WETH,
  amountIn: "1000",
  slippage: 0.5,
  maxHops:  2,
  network:  NETWORK.BASE,
  rpcUrls:  [{ url: "https://rpc.example.com" }],
}

const successData = {
  tokenIn:       USDC,
  tokenOut:      WETH,
  amountIn:      "1000",
  amountOut:     "0.5",
  minOut:        "0.4975",
  amountInRaw:   "1000000000",
  amountOutRaw:  "500000000000000000",
  minOutRaw:     "497500000000000000",
  steps:         "0xabcdef",
  slippage:      0.5,
  path:          [USDC, WETH],
  hops: [
    {
      tokenIn:       USDC,
      tokenOut:      WETH,
      amountIn:      "1000",
      amountOut:     "0.5",
      minOut:        "0.4975",
      amountInRaw:   "1000000000",
      amountOutRaw:  "500000000000000000",
      minOutRaw:     "497500000000000000",
      tokenInPrice:  "0.00047147437880193935",
      tokenOutPrice: "2121.006029089203",
      slippage:      0.5,
      type:          "uniV3",
      kind:          "uniswap",
      routeId:       1,
      weight:        7.65,
    },
  ],
  tokenInPrice:  "0.00047147437880193935",
  tokenOutPrice: "2121.006029089203",
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

    const quote = await fetchQuote(baseParams, 35, QUOTER_URL)

    expect(quote.tokenIn).toBe(USDC)
    expect(quote.tokenOut).toBe(WETH)
    expect(quote.amountIn).toBe("1000")
    expect(quote.amountOut).toBe("0.5")
    expect(quote.minOut).toBe("0.4975")
    expect(quote.amountInWei).toBe(1000000000n)
    expect(quote.amountOutWei).toBe(500000000000000000n)
    expect(quote.minOutWei).toBe(497500000000000000n)
    expect(quote.steps).toBe("0xabcdef")
    expect(quote.slippage).toBe(0.5)
    expect(quote.feeBps).toBe(35)
    expect(quote.path).toEqual([USDC, WETH])
    expect(quote.hops).toHaveLength(1)
    expect(quote.hops[0].type).toBe("uniV3")
    expect(quote.hops[0].kind).toBe("uniswap")
    expect(quote.hops[0].amountOut).toBe("0.5")
    expect(quote.tokenInPrice).toBe("0.00047147437880193935")
    expect(quote.tokenOutPrice).toBe("2121.006029089203")
  })

  it("sends correct request body to quoter", async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify({ status: "success", data: successData }), { status: 200 }),
    )

    await fetchQuote(baseParams, 35, QUOTER_URL)

    const [url, options] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe(QUOTER_URL)

    const body = JSON.parse(options.body as string)
    expect(body.network).toBe("base")
    expect(body.tokenIn).toBe(USDC)
    expect(body.tokenOut).toBe(WETH)
    expect(body.amountIn).toBe("1000")
    expect(body.slippage).toBe(0.5)
    expect(body.maxHops).toBe(2)
    expect(body.show).toBe(true)
    expect(body.rpcUrls).toEqual([{ url: "https://rpc.example.com" }])
  })

  it("sends BSC network when set", async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify({ status: "success", data: successData }), { status: 200 }),
    )

    await fetchQuote({ ...baseParams, network: NETWORK.BSC }, 35, QUOTER_URL)

    const [, options] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    const body = JSON.parse(options.body as string)
    expect(body.network).toBe("bsc")
  })

  it("includes priceBase in body when set", async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify({ status: "success", data: successData }), { status: 200 }),
    )

    await fetchQuote({ ...baseParams, priceBase: "USDC" }, 35, QUOTER_URL)

    const [, options] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    const body = JSON.parse(options.body as string)
    expect(body.priceBase).toBe("USDC")
  })

  it("includes dexs in body when set", async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify({ status: "success", data: successData }), { status: 200 }),
    )

    await fetchQuote({ ...baseParams, dexs: [DEX.UNI_V3, DEX.AERODROME] }, 35, QUOTER_URL)

    const [, options] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    const body = JSON.parse(options.body as string)
    expect(body.dexs).toEqual(["uni-v3", "aerodrome"])
  })

  it("omits priceBase and dexs from body when not set", async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify({ status: "success", data: successData }), { status: 200 }),
    )

    await fetchQuote(baseParams, 35, QUOTER_URL)

    const [, options] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    const body = JSON.parse(options.body as string)
    expect(body.priceBase).toBeUndefined()
    expect(body.dexs).toBeUndefined()
  })

  it("maps tokenInBasePrice and tokenOutBasePrice from response", async () => {
    const dataWithBase = { ...successData, tokenInBasePrice: "1.00", tokenOutBasePrice: "2121.00" }
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify({ status: "success", data: dataWithBase }), { status: 200 }),
    )

    const quote = await fetchQuote(baseParams, 35, QUOTER_URL)

    expect(quote.tokenInBasePrice).toBe("1.00")
    expect(quote.tokenOutBasePrice).toBe("2121.00")
  })

  it("leaves tokenInBasePrice undefined when not in response", async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify({ status: "success", data: successData }), { status: 200 }),
    )

    const quote = await fetchQuote(baseParams, 35, QUOTER_URL)

    expect(quote.tokenInBasePrice).toBeUndefined()
    expect(quote.tokenOutBasePrice).toBeUndefined()
  })

  it("sends amountIn string directly to quoter", async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify({ status: "success", data: successData }), { status: 200 }),
    )

    await fetchQuote({ ...baseParams, amountIn: "1.5" }, 35, QUOTER_URL)

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

    await expect(fetchQuote(baseParams, 35, QUOTER_URL)).rejects.toThrow("no route found")
  })

  it("throws QuoteError when minOutRaw is zero", async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(
        JSON.stringify({ status: "success", data: { ...successData, minOutRaw: "0" } }),
        { status: 200 },
      ),
    )

    await expect(fetchQuote(baseParams, 35, QUOTER_URL)).rejects.toThrow("zero minOut")
  })

  it("throws QuoteError on HTTP error", async () => {
    vi.mocked(fetch).mockResolvedValue(new Response("", { status: 502 }))

    await expect(fetchQuote(baseParams, 35, QUOTER_URL)).rejects.toThrow("HTTP 502")
  })

  it("throws QuoteError on network failure", async () => {
    vi.mocked(fetch).mockRejectedValue(new Error("ECONNREFUSED"))

    await expect(fetchQuote(baseParams, 35, QUOTER_URL)).rejects.toThrow("ECONNREFUSED")
  })

  it("uses fallback message when error has no message field", async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify({ status: "error" }), { status: 200 }),
    )

    await expect(fetchQuote(baseParams, 35, QUOTER_URL)).rejects.toThrow(
      "unknown error from quoter",
    )
  })

  it("extracts message from string data on error", async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(
        JSON.stringify({ status: "error", data: "insufficient liquidity" }),
        { status: 200 },
      ),
    )

    await expect(fetchQuote(baseParams, 35, QUOTER_URL)).rejects.toThrow("insufficient liquidity")
  })
})
