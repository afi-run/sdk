import { afterEach, beforeEach, describe, expect, it, vi, type MockInstance } from "vitest"
import { QuoteBuilder } from "../builder.js"
import * as quoter from "../quoter.js"
import { DEX, NETWORK, type Quote, type SwapResult } from "../types.js"

const USDC = "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913" as const
const WETH = "0x4200000000000000000000000000000000000006" as const

const fakeQuote = { tokenIn: USDC, tokenOut: WETH, amountIn: "1000" } as unknown as Quote
const fakeResult = { txHash: "0xabc" } as unknown as SwapResult

interface FakeClient {
  rpcUrl: string
  quoterUrl: string
  getFeeBps: ReturnType<typeof vi.fn>
  executeSwap: ReturnType<typeof vi.fn>
  _runLogged: <T>(method: string, fn: () => Promise<T>) => Promise<T>
}

let client: FakeClient
let fetchQuoteSpy: MockInstance<typeof quoter.fetchQuote>

beforeEach(() => {
  client = {
    rpcUrl:      "https://rpc.example.com",
    quoterUrl:   "https://api.example.com/quoter",
    getFeeBps:   vi.fn().mockResolvedValue(35),
    executeSwap: vi.fn().mockResolvedValue(fakeResult),
    _runLogged:  <T>(_m: string, fn: () => Promise<T>) => fn(),
  }
  fetchQuoteSpy = vi.spyOn(quoter, "fetchQuote").mockResolvedValue(fakeQuote)
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe("QuoteBuilder", () => {
  it("seeds defaults: slippage=0.5, maxHops=2, network=BASE", async () => {
    await new QuoteBuilder(client as any, USDC, WETH, "1000").get()

    const req = fetchQuoteSpy.mock.calls[0][0]
    expect(req.slippage).toBe(0.5)
    expect(req.maxHops).toBe(2)
    expect(req.network).toBe(NETWORK.BASE)
    expect(req.amountIn).toBe("1000")
  })

  it("resolves Address string for tokenIn and tokenOut", async () => {
    await new QuoteBuilder(client as any, USDC, WETH, "1000").get()
    const req = fetchQuoteSpy.mock.calls[0][0]
    expect(req.tokenIn).toBe(USDC)
    expect(req.tokenOut).toBe(WETH)
  })

  it("resolves Token objects for tokenIn and tokenOut", async () => {
    const tokenIn = { address: USDC, symbol: "USDC", decimals: 6, active: true }
    const tokenOut = { address: WETH, symbol: "WETH", decimals: 18, active: true }
    await new QuoteBuilder(client as any, tokenIn, tokenOut, "1000").get()

    const req = fetchQuoteSpy.mock.calls[0][0]
    expect(req.tokenIn).toBe(USDC)
    expect(req.tokenOut).toBe(WETH)
  })

  it("slippage() overrides default", async () => {
    await new QuoteBuilder(client as any, USDC, WETH, "1000").slippage(1.25).get()
    expect(fetchQuoteSpy.mock.calls[0][0].slippage).toBe(1.25)
  })

  it("maxHops() overrides default", async () => {
    await new QuoteBuilder(client as any, USDC, WETH, "1000").maxHops(5).get()
    expect(fetchQuoteSpy.mock.calls[0][0].maxHops).toBe(5)
  })

  it("network() overrides default", async () => {
    await new QuoteBuilder(client as any, USDC, WETH, "1000").network(NETWORK.BSC).get()
    expect(fetchQuoteSpy.mock.calls[0][0].network).toBe(NETWORK.BSC)
  })

  it("priceBase() sets priceBase", async () => {
    await new QuoteBuilder(client as any, USDC, WETH, "1000").priceBase("USDC").get()
    expect(fetchQuoteSpy.mock.calls[0][0].priceBase).toBe("USDC")
  })

  it("dexs() sets dex list", async () => {
    await new QuoteBuilder(client as any, USDC, WETH, "1000")
      .dexs(DEX.UNI_V3, DEX.AERODROME)
      .get()
    expect(fetchQuoteSpy.mock.calls[0][0].dexs).toEqual([DEX.UNI_V3, DEX.AERODROME])
  })

  it("blockNumber() sets blockNumber", async () => {
    await new QuoteBuilder(client as any, USDC, WETH, "1000").blockNumber("0x1234").get()
    expect(fetchQuoteSpy.mock.calls[0][0].blockNumber).toBe("0x1234")
  })

  it("rpcUrls() sets rpcUrls and skips client fallback", async () => {
    const custom = [{ url: "https://custom.rpc" }]
    await new QuoteBuilder(client as any, USDC, WETH, "1000").rpcUrls(...custom).get()
    expect(fetchQuoteSpy.mock.calls[0][0].rpcUrls).toEqual(custom)
  })

  it("falls back to client.rpcUrl when no rpcUrls configured", async () => {
    await new QuoteBuilder(client as any, USDC, WETH, "1000").get()
    expect(fetchQuoteSpy.mock.calls[0][0].rpcUrls).toEqual([{ url: client.rpcUrl }])
  })

  it("returns chainable instance", () => {
    const b = new QuoteBuilder(client as any, USDC, WETH, "1000")
    expect(b.slippage(1)).toBe(b)
    expect(b.maxHops(3)).toBe(b)
    expect(b.network(NETWORK.BSC)).toBe(b)
    expect(b.priceBase("USDC")).toBe(b)
    expect(b.dexs(DEX.UNI_V3)).toBe(b)
    expect(b.blockNumber("latest")).toBe(b)
    expect(b.rpcUrls({ url: "x" })).toBe(b)
  })

  it("get() passes feeBps and quoterUrl from client to fetchQuote", async () => {
    await new QuoteBuilder(client as any, USDC, WETH, "1000").get()
    expect(client.getFeeBps).toHaveBeenCalled()
    expect(fetchQuoteSpy).toHaveBeenCalledWith(expect.any(Object), 35, client.quoterUrl)
  })

  it("execute() calls client.executeSwap with the fetched quote", async () => {
    const result = await new QuoteBuilder(client as any, USDC, WETH, "1000").execute()
    expect(client.executeSwap).toHaveBeenCalledWith(fakeQuote)
    expect(result).toBe(fakeResult)
  })
})
