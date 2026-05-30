import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { AfiClient } from "../client.js"
import type { LogEvent } from "../types.js"

const PK = "0x" + "11".repeat(32) as `0x${string}`

describe("AfiClient — logger", () => {
  let events: LogEvent[]
  let client: AfiClient

  beforeEach(() => {
    events = []
    client = new AfiClient({
      rpcUrl: "http://localhost:1",
      logger: (e) => events.push(e),
    })
  })

  it("fires an ok event on successful logged operation", async () => {
    // simulate is the simplest signer-only method; we'll patch the pub client to make it pass
    client.connect(PK)
    ;(client as any).pub.simulateContract = vi.fn().mockResolvedValue({})

    await client.simulate({ tokenIn: "0x0", tokenOut: "0x0", amountInWei: 0n, minOutWei: 0n, steps: "0x" } as any)

    expect(events).toHaveLength(1)
    expect(events[0].method).toBe("simulate")
    expect(events[0].ok).toBe(true)
    expect(events[0].durationMs).toBeGreaterThanOrEqual(0)
  })

  it("fires a failure event with the error attached", async () => {
    client.connect(PK)
    const boom = new Error("simulated failure")
    ;(client as any).pub.simulateContract = vi.fn().mockRejectedValue(boom)

    await expect(
      client.simulate({ tokenIn: "0x0", tokenOut: "0x0", amountInWei: 0n, minOutWei: 0n, steps: "0x" } as any),
    ).rejects.toBeDefined()

    expect(events).toHaveLength(1)
    expect(events[0].ok).toBe(false)
    expect(events[0].error).toBeInstanceOf(Error)
  })

  it("setLogger(undefined) disables future emissions", async () => {
    client.connect(PK)
    ;(client as any).pub.simulateContract = vi.fn().mockResolvedValue({})

    client.setLogger(undefined)
    await client.simulate({ tokenIn: "0x0", tokenOut: "0x0", amountInWei: 0n, minOutWei: 0n, steps: "0x" } as any)

    expect(events).toHaveLength(0)
  })
})

describe("AfiClient — chainId / detectNetwork", () => {
  it("caches the chain ID across calls", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    const spy = vi.fn().mockResolvedValue(8453)
    ;(client as any).pub.getChainId = spy

    expect(await client.chainId()).toBe(8453)
    expect(await client.chainId()).toBe(8453)
    expect(spy).toHaveBeenCalledTimes(1)
  })

  it("detectNetwork maps a known chain ID to a Network", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    ;(client as any).pub.getChainId = vi.fn().mockResolvedValue(56)
    expect(await client.detectNetwork()).toBe("bsc")
  })

  it("detectNetwork returns null for unknown chain", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    ;(client as any).pub.getChainId = vi.fn().mockResolvedValue(999_999)
    expect(await client.detectNetwork()).toBeNull()
  })
})

describe("AfiClient — waitForTx", () => {
  it("forwards confirmations and timeout to viem", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    const spy = vi.fn().mockResolvedValue({ blockNumber: 42n, gasUsed: 21000n })
    ;(client as any).pub.waitForTransactionReceipt = spy

    const receipt = await client.waitForTx("0xabc" as `0x${string}`, { confirmations: 3, timeoutMs: 10_000 })

    expect(receipt.blockNumber).toBe(42n)
    expect(receipt.gasUsed).toBe(21000n)
    expect(spy).toHaveBeenCalledWith(expect.objectContaining({
      hash: "0xabc",
      confirmations: 3,
      timeout: 10_000,
    }))
  })
})

describe("AfiClient — config defaults", () => {
  afterEach(() => vi.restoreAllMocks())

  it("logger is optional", () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    expect(client.gasBufferPercent).toBe(15)
  })
})

describe("AfiClient — estimateSwapCost", () => {
  it("multiplies gas by buffer and combines with maxFeePerGas", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1", gasBufferPercent: 20 })
    client.connect(PK)
    ;(client as any).pub.estimateContractGas = vi.fn().mockResolvedValue(100_000n)
    ;(client as any).pub.getBlock = vi.fn().mockResolvedValue({ baseFeePerGas: 1_000_000_000n })
    ;(client as any).pub.estimateMaxPriorityFeePerGas = vi.fn().mockResolvedValue(500_000_000n)

    const est = await client.estimateSwapCost({
      tokenIn: "0x0", tokenOut: "0x0", amountInWei: 0n, minOutWei: 0n, steps: "0x",
    } as any)

    expect(est.gas).toBe(100_000n)
    expect(est.gasWithBuffer).toBe(120_000n)               // +20%
    expect(est.gasPriceWei).toBe(2_500_000_000n)            // baseFee*2 + tip
    expect(est.totalWei).toBe(120_000n * 2_500_000_000n)
    expect(est.totalEth).toMatch(/^0\.0003/)
  })

  it("falls back to 0 tip when estimateMaxPriorityFeePerGas reverts", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1", gasBufferPercent: 0 })
    client.connect(PK)
    ;(client as any).pub.estimateContractGas = vi.fn().mockResolvedValue(50_000n)
    ;(client as any).pub.getBlock = vi.fn().mockResolvedValue({ baseFeePerGas: 1_000_000_000n })
    ;(client as any).pub.estimateMaxPriorityFeePerGas = vi.fn().mockRejectedValue(new Error("eip-1559 not supported"))

    const est = await client.estimateSwapCost({
      tokenIn: "0x0", tokenOut: "0x0", amountInWei: 0n, minOutWei: 0n, steps: "0x",
    } as any)
    expect(est.gasPriceWei).toBe(2_000_000_000n) // baseFee*2 + 0
  })

  it("throws NoSignerError without a signer", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    await expect(
      client.estimateSwapCost({ tokenIn: "0x0", tokenOut: "0x0", amountInWei: 0n, minOutWei: 0n, steps: "0x" } as any),
    ).rejects.toMatchObject({ code: "NO_SIGNER" })
  })
})

describe("AfiClient — health", () => {
  it("returns ok=true when both endpoints respond", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    ;(client as any).pub.getChainId = vi.fn().mockResolvedValue(8453)
    const fetchSpy = vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response("", { status: 200 }))

    const h = await client.health()

    expect(h.rpc.ok).toBe(true)
    expect(h.rpc.detail).toBe("chainId=8453")
    expect(h.api.ok).toBe(true)
    expect(h.api.detail).toBe("ok")
    fetchSpy.mockRestore()
  })

  it("reports rpc failure without aborting api probe", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    ;(client as any).pub.getChainId = vi.fn().mockRejectedValue(new Error("ECONNREFUSED"))
    const fetchSpy = vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response("", { status: 200 }))

    const h = await client.health()

    expect(h.rpc.ok).toBe(false)
    expect(h.rpc.error).toBeInstanceOf(Error)
    expect(h.api.ok).toBe(true)
    fetchSpy.mockRestore()
  })

  it("reports api HTTP error code", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    ;(client as any).pub.getChainId = vi.fn().mockResolvedValue(8453)
    const fetchSpy = vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response("", { status: 503 }))

    const h = await client.health()

    expect(h.api.ok).toBe(false)
    expect(h.api.detail).toBe("HTTP 503")
    fetchSpy.mockRestore()
  })
})

describe("AfiClient — tokenInfo metadata cache", () => {
  it("caches symbol/name/decimals across calls (no owner)", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    const TOKEN = "0xaaaa589fcd6edb6e08f4c7c32d4f71b54bda02913" as const

    const mcSpy = vi.fn().mockResolvedValue([
      { status: "success", result: "USDC" },
      { status: "success", result: "USD Coin" },
      { status: "success", result: 6 },
    ])
    ;(client as any).pub.multicall = mcSpy

    await client.tokenInfo(TOKEN)
    await client.tokenInfo(TOKEN)
    await client.tokenInfo(TOKEN)
    // Second and third calls should not hit the multicall at all because
    // metadata is cached and no owner is requested.
    expect(mcSpy).toHaveBeenCalledTimes(1)
  })

  it("with owner, reuses cached metadata and only fetches balance/allowance", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    client.connect(PK)
    const TOKEN = "0xbbbb589fcd6edb6e08f4c7c32d4f71b54bda02913" as const

    const mc = vi.fn()
      .mockResolvedValueOnce([
        { status: "success", result: "USDC" },
        { status: "success", result: "USD Coin" },
        { status: "success", result: 6 },
        { status: "success", result: 1_000n },
        { status: "success", result: 500n },
      ])
      .mockResolvedValueOnce([
        // Only balance + allowance (metadata cached)
        { status: "success", result: 2_000n },
        { status: "success", result: 700n },
      ])
    ;(client as any).pub.multicall = mc

    await client.tokenInfo(TOKEN, "self")
    await client.tokenInfo(TOKEN, "self")

    expect(mc.mock.calls[1][0].contracts).toHaveLength(2)
    expect(mc.mock.calls[1][0].contracts[0].functionName).toBe("balanceOf")
    expect(mc.mock.calls[1][0].contracts[1].functionName).toBe("allowance")
  })

  it("clearTokenMetadataCache forces a refetch", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    const TOKEN = "0xcccc589fcd6edb6e08f4c7c32d4f71b54bda02913" as const
    const mc = vi.fn().mockResolvedValue([
      { status: "success", result: "X" },
      { status: "success", result: "X" },
      { status: "success", result: 18 },
    ])
    ;(client as any).pub.multicall = mc

    await client.tokenInfo(TOKEN)
    client.clearTokenMetadataCache()
    await client.tokenInfo(TOKEN)

    expect(mc).toHaveBeenCalledTimes(2)
  })
})

describe("AfiClient — multicall generic", () => {
  it("forwards contracts to the Multicall3 address", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    const spy = vi.fn().mockResolvedValue([{ status: "success", result: 42n }])
    ;(client as any).pub.multicall = spy

    const TOKEN = "0xaaaa589fcd6edb6e08f4c7c32d4f71b54bda02913" as const
    const results = await client.multicall([
      { address: TOKEN, abi: [], functionName: "totalSupply" },
    ])

    expect(results).toHaveLength(1)
    expect(results[0].status).toBe("success")
    expect(spy.mock.calls[0][0].multicallAddress).toBeDefined()
    expect(spy.mock.calls[0][0].allowFailure).toBe(true)
  })

  it("respects allowFailure override", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    const spy = vi.fn().mockResolvedValue([])
    ;(client as any).pub.multicall = spy

    await client.multicall([], { allowFailure: false })
    expect(spy.mock.calls[0][0].allowFailure).toBe(false)
  })

  it("maps failure entries to {status:failure, error}", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    const err = new Error("revert XYZ")
    ;(client as any).pub.multicall = vi.fn().mockResolvedValue([
      { status: "failure", error: err },
    ])
    const r = await client.multicall([{ address: "0x0", abi: [], functionName: "x" } as any])
    expect(r[0].status).toBe("failure")
    if (r[0].status === "failure") expect(r[0].error).toBe(err)
  })
})

describe("AfiClient — revoke", () => {
  const TOKEN = "0xdddd589fcd6edb6e08f4c7c32d4f71b54bda02913" as const

  it("returns null when allowance is already zero", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    client.connect(PK)
    ;(client as any).pub.readContract = vi.fn().mockResolvedValue(0n)
    const r = await client.revoke(TOKEN)
    expect(r).toBeNull()
  })

  it("sends approve(AFI, 0) when allowance is non-zero", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    client.connect(PK)
    ;(client as any).pub.readContract = vi.fn().mockResolvedValue(1_000n)
    ;(client as any).pub.estimateContractGas = vi.fn().mockResolvedValue(40_000n)
    const writeSpy = vi.fn().mockResolvedValue("0xrevoke")
    ;(client as any).wallet = { writeContract: writeSpy }

    const r = await client.revoke(TOKEN)
    expect(r).not.toBeNull()
    expect(r!.txHash).toBe("0xrevoke")
    expect(writeSpy.mock.calls[0][0].args[1]).toBe(0n)
  })
})

describe("AfiClient — refreshQuote", () => {
  it("rebuilds the builder with the same network / slippage / maxHops / dexs", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    const stale: any = {
      tokenIn:  "0xaaaa",
      tokenOut: "0xbbbb",
      amountIn: "1000",
      slippage: 1.25,
      network:  "bsc",
      maxHops:  3,
      priceBase: "USDC",
      dexs:     ["uni-v3", "cake-v3"],
    }
    const fresh = { ...stale, amountOut: "fresh", createdAt: Date.now() }
    const builderGet = vi.fn().mockResolvedValue(fresh)
    const builder: any = {
      slippage: vi.fn().mockReturnThis(),
      network:  vi.fn().mockReturnThis(),
      maxHops:  vi.fn().mockReturnThis(),
      priceBase: vi.fn().mockReturnThis(),
      dexs:      vi.fn().mockReturnThis(),
      get:       builderGet,
    }
    ;(client as any).quote = vi.fn().mockReturnValue(builder)

    const out = await client.refreshQuote(stale)
    expect(out).toBe(fresh)
    expect(builder.slippage).toHaveBeenCalledWith(1.25)
    expect(builder.network).toHaveBeenCalledWith("bsc")
    expect(builder.maxHops).toHaveBeenCalledWith(3)
    expect(builder.priceBase).toHaveBeenCalledWith("USDC")
    expect(builder.dexs).toHaveBeenCalledWith("uni-v3", "cake-v3")
  })
})

describe("AfiClient — confirmations passthrough", () => {
  it("forwards executeSwap confirmations to pending.wait", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    client.connect(PK)
    const waitSpy = vi.fn().mockResolvedValue({ txHash: "0x", blockNumber: 1n, amountIn: 0n, amountOut: 0n, tokenIn: "0x", tokenOut: "0x", gasUsed: 0n })

    // Short-circuit every internal step
    ;(client as any).pub.readContract = vi.fn().mockResolvedValue(10_000_000n) // balance+allowance OK
    ;(client as any).pub.simulateContract = vi.fn().mockResolvedValue({})
    ;(client as any).pub.estimateContractGas = vi.fn().mockResolvedValue(100_000n)
    ;(client as any).wallet = { writeContract: vi.fn().mockResolvedValue("0xabc") }

    // Stub sendSwap by intercepting submitSwap path: we patch waitForTransactionReceipt to capture forwarded opts.
    const wftrSpy = vi.fn().mockResolvedValue({ logs: [], blockNumber: 1n, gasUsed: 0n })
    ;(client as any).pub.waitForTransactionReceipt = wftrSpy

    await client.executeSwap(
      { tokenIn: "0x0", tokenOut: "0x0", amountInWei: 1_000_000n, minOutWei: 1n, steps: "0x" } as any,
      { confirmations: 3, timeoutMs: 5_000 },
    ).catch(() => {/* parseEventLogs will fail; we only care about the wait call */})

    expect(wftrSpy).toHaveBeenCalledWith(expect.objectContaining({
      confirmations: 3,
      timeout: 5_000,
    }))
  })
})
