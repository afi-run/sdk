import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import {
  fetchQuote,
  findArbitrage,
  findPath,
  getLiquidationCandidates,
  getRoutes,
  liquidate,
  priceQuote,
  quoteDex,
  quoteFromRoute,
  routeProfit,
  type QuoteRequest,
  type RouteQuote,
} from "../quoter.js"
import { BadRequestError, NetworkError, QuoteError, ServerError } from "../errors.js"
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

// ─── New afi-rpc HTTP endpoints ──────────────────────────────────────────────

const API = "https://rpc.afi.run"

function mockSuccess(data: unknown): void {
  vi.mocked(fetch).mockResolvedValue(
    new Response(JSON.stringify({ status: "success", data }), { status: 200 }),
  )
}

const sampleRoute: RouteQuote = {
  network: "base",
  tokenIn: USDC,
  tokenOut: USDC,
  amountIn: "1000",
  amountInRaw: "1000000000",
  amountOut: "1005",
  amountOutRaw: "1005000000",
  minOut: "1000",
  minOutRaw: "1000000000",
  routeId: 3,
  stepData: "0xdeadbeef",
}

describe("routeProfit", () => {
  it("returns amountOutRaw − amountInRaw", () => {
    expect(routeProfit(sampleRoute)).toBe(5_000_000n)
  })
  it("returns null on unparseable amounts", () => {
    expect(routeProfit({ ...sampleRoute, amountInRaw: "x" })).toBeNull()
  })
})

describe("quoteFromRoute", () => {
  it("hydrates an executable Quote from a route", () => {
    const q = quoteFromRoute(sampleRoute)
    expect(q.tokenIn).toBe(q.tokenOut) // cycle
    expect(q.amountInWei).toBe(1_000_000_000n)
    expect(q.minOutWei).toBe(1_000_000_000n)
    expect(q.network).toBe("base")
    // Steps = encodeSteps of the single hop: numSteps(1) + id(0003) + len(0004) + deadbeef
    expect(q.steps).toBe("0x0100030004deadbeef")
  })
  it("honours minOutOverride", () => {
    const q = quoteFromRoute(sampleRoute, 1_002_000_000n)
    expect(q.minOutWei).toBe(1_002_000_000n)
  })
  it("rejects a routeId outside uint16 range", () => {
    expect(() => quoteFromRoute({ ...sampleRoute, routeId: 70000 })).toThrow(/uint16/)
  })
})

describe("findArbitrage", () => {
  it("POSTs /arbitrage and returns route quotes", async () => {
    mockSuccess([
      {
        network: "base", tokenIn: USDC, tokenOut: USDC,
        amountIn: "1", amountInRaw: "1000000",
        amountOut: "1.01", amountOutRaw: "1010000",
        minOutRaw: "1000000", routeId: 3, stepData: "0xabcd",
      },
    ])
    const res = await findArbitrage(API, { network: NETWORK.BASE, tokenIn: USDC, tokenOut: USDC, amountIn: "1" })
    expect(res).toHaveLength(1)
    expect(res[0].routeId).toBe(3)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe(`${API}/arbitrage`)
    const body = JSON.parse(opts.body as string)
    expect(body.tokenIn).toBe(USDC)
  })

  it("throws on HTTP 500", async () => {
    vi.mocked(fetch).mockResolvedValue(new Response("", { status: 500 }))
    await expect(
      findArbitrage(API, { network: NETWORK.BASE, tokenIn: USDC, tokenOut: USDC, amountIn: "1" }),
    ).rejects.toThrow("HTTP 500")
  })

  const arb = { network: NETWORK.BASE, tokenIn: USDC, tokenOut: USDC, amountIn: "1" }

  it("surfaces json.message on a non-success body", async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify({ status: "error", message: "no liquidity" }), { status: 200 }),
    )
    await expect(findArbitrage(API, arb)).rejects.toThrow("no liquidity")
  })

  it("uses string data as the message when there is no message field", async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify({ status: "error", data: "rate limited" }), { status: 200 }),
    )
    await expect(findArbitrage(API, arb)).rejects.toThrow("rate limited")
  })

  it("falls back to a generic message when neither message nor string data is present", async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify({ status: "error" }), { status: 200 }),
    )
    await expect(findArbitrage(API, arb)).rejects.toThrow("unknown error from afi-rpc")
  })

  it("treats a success status with undefined data as an error", async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify({ status: "success" }), { status: 200 }),
    )
    await expect(findArbitrage(API, arb)).rejects.toThrow("unknown error from afi-rpc")
  })
})

describe("findPath", () => {
  it("POSTs /command with action=path and returns a priced route", async () => {
    mockSuccess({
      network: "base", path: [USDC, WETH], tokenIn: USDC, tokenOut: WETH,
      amountIn: "1", amountInRaw: "1000000", amountOut: "0.0005", amountOutRaw: "500000000000000",
      minOut: "0.0004", minOutRaw: "400000000000000", steps: "0xbeef", hops: [],
    })
    const res = await findPath(API, { network: NETWORK.BASE, tokenIn: USDC, tokenOut: WETH })
    expect(res.path).toEqual([USDC, WETH])
    expect(res.steps).toBe("0xbeef")
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe(`${API}/command`)
    expect(JSON.parse(opts.body as string).action).toBe("path")
  })
})

describe("getRoutes", () => {
  it("POSTs /command with action=routes and returns token paths", async () => {
    mockSuccess([{ path: [USDC, WETH] }, { path: [USDC, WETH, USDC] }])
    const res = await getRoutes(API, { network: NETWORK.BASE })
    expect(res).toHaveLength(2)
    expect(res[0].path).toEqual([USDC, WETH])
    const [, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(JSON.parse(opts.body as string).action).toBe("routes")
  })
})

describe("priceQuote", () => {
  it("POSTs /command with action=price and returns route quotes", async () => {
    mockSuccess([
      { network: "base", tokenIn: USDC, tokenOut: WETH, amountIn: "1", amountInRaw: "1000000", amountOut: "0.0005", amountOutRaw: "500000000000000", minOutRaw: "0", routeId: 3, stepData: "0x" },
    ])
    const res = await priceQuote(API, { network: NETWORK.BASE, tokenIn: USDC, tokenOut: WETH, amountIn: "1" })
    expect(res[0].routeId).toBe(3)
    const [, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(JSON.parse(opts.body as string).action).toBe("price")
  })
})

describe("quoteDex", () => {
  it.each(["uniV3", "cakeV3", "uniV4", "aerodrome", "balancer", "fluid", "curve128", "curve256"] as const)(
    "POSTs /command with action=%s",
    async (dex) => {
      mockSuccess([{ network: "base", tokenIn: USDC, tokenOut: WETH, amountIn: "1", amountInRaw: "1000000", amountOut: "1", amountOutRaw: "1000000", minOutRaw: "0", routeId: 3, stepData: "0x" }])
      const res = await quoteDex(API, dex, {
        network: NETWORK.BASE,
        tokenIn: USDC,
        tokenOut: WETH,
        amountIn: "1",
      })
      expect(res[0].routeId).toBe(3)
      const [, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
      expect(JSON.parse(opts.body as string).action).toBe(dex)
    },
  )
})

describe("getLiquidationCandidates", () => {
  it("POSTs /aave and returns Aave positions", async () => {
    mockSuccess([{ user: USDC, debtToken: "USDC", debtAmount: "500", collaterals: [{ token: "WETH", balance: "1.5" }] }])
    const list = await getLiquidationCandidates(API, { network: NETWORK.BASE })
    expect(list).toHaveLength(1)
    expect(list[0].collaterals).toHaveLength(1)
    const [url] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe(`${API}/aave`)
  })
})

describe("liquidate", () => {
  it("POSTs /liquidation-call and returns a repay+swap route", async () => {
    mockSuccess({
      tokenIn: USDC, tokenOut: WETH, amountIn: "1", amountOut: "1.05",
      profit: "0.05", steps: "0xbeef",
      hops: [{ routeId: 10, kind: "aave" }, { routeId: 3, kind: "uni" }],
    })
    const res = await liquidate(API, {
      network: NETWORK.BASE, pool: USDC, user: USDC,
      tokenIn: USDC, tokenOut: WETH, amountIn: "1",
    })
    expect(res.profit).toBe("0.05")
    expect(res.hops).toHaveLength(2)
    const [url] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe(`${API}/liquidation-call`)
  })
})

describe("typed HTTP errors", () => {
  it("fetchQuote throws BadRequestError on 4xx", async () => {
    vi.mocked(fetch).mockResolvedValue(new Response("", { status: 400 }))
    const p = fetchQuote(baseParams, 35, QUOTER_URL)
    await expect(p).rejects.toBeInstanceOf(BadRequestError)
    // Also catchable as the base class.
    await expect(p).rejects.toBeInstanceOf(QuoteError)
  })

  it("fetchQuote throws ServerError on 5xx", async () => {
    vi.mocked(fetch).mockResolvedValue(new Response("", { status: 500 }))
    const p = fetchQuote(baseParams, 35, QUOTER_URL)
    await expect(p).rejects.toBeInstanceOf(ServerError)
    await expect(p).rejects.toBeInstanceOf(QuoteError)
  })

  it("fetchQuote throws NetworkError on fetch rejection", async () => {
    vi.mocked(fetch).mockRejectedValue(new Error("ECONNREFUSED"))
    const p = fetchQuote(baseParams, 35, QUOTER_URL)
    await expect(p).rejects.toBeInstanceOf(NetworkError)
    await expect(p).rejects.toBeInstanceOf(QuoteError)
  })

  it("postJson-style endpoint throws BadRequestError on 4xx", async () => {
    vi.mocked(fetch).mockResolvedValue(new Response("", { status: 404 }))
    await expect(
      findArbitrage(API, { network: NETWORK.BASE, tokenIn: USDC, tokenOut: USDC, amountIn: "1" }),
    ).rejects.toBeInstanceOf(BadRequestError)
  })

  it("postJson-style endpoint throws ServerError on 5xx", async () => {
    vi.mocked(fetch).mockResolvedValue(new Response("", { status: 503 }))
    await expect(
      findArbitrage(API, { network: NETWORK.BASE, tokenIn: USDC, tokenOut: USDC, amountIn: "1" }),
    ).rejects.toBeInstanceOf(ServerError)
  })

  it("postJson-style endpoint throws NetworkError on fetch rejection", async () => {
    vi.mocked(fetch).mockRejectedValue(new Error("DNS fail"))
    await expect(
      findArbitrage(API, { network: NETWORK.BASE, tokenIn: USDC, tokenOut: USDC, amountIn: "1" }),
    ).rejects.toBeInstanceOf(NetworkError)
  })
})
