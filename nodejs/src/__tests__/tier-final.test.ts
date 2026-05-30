import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { AfiClient } from "../client.js"
import { feeFromReceipt } from "../swap.js"

const PK = "0x" + "11".repeat(32) as `0x${string}`

describe("feeFromReceipt", () => {
  it("computes wei × gas and ETH formatted output", () => {
    const fee = feeFromReceipt(150_000n, 2_000_000_000n) // 0.0003 ETH
    expect(fee.effectiveGasPrice).toBe(2_000_000_000n)
    expect(fee.feeWei).toBe(300_000_000_000_000n)
    expect(fee.feeEth).toBe("0.0003")
  })

  it("defaults effectiveGasPrice to 0n when receipt missing", () => {
    const fee = feeFromReceipt(150_000n, undefined)
    expect(fee.effectiveGasPrice).toBe(0n)
    expect(fee.feeWei).toBe(0n)
    expect(fee.feeEth).toBe("0")
  })
})

describe("AfiClient — getTxStatus", () => {
  let client: AfiClient
  beforeEach(() => { client = new AfiClient({ rpcUrl: "http://localhost:1" }) })
  afterEach(() => vi.restoreAllMocks())

  it("returns 'success' when the receipt has status='success'", async () => {
    ;(client as any).pub.getTransactionReceipt = vi.fn().mockResolvedValue({ status: "success" })
    expect(await client.getTxStatus("0xabc" as `0x${string}`)).toBe("success")
  })

  it("returns 'failed' when the receipt status is anything else", async () => {
    ;(client as any).pub.getTransactionReceipt = vi.fn().mockResolvedValue({ status: "reverted" })
    expect(await client.getTxStatus("0xabc" as `0x${string}`)).toBe("failed")
  })

  it("returns 'pending' when receipt is missing but tx exists", async () => {
    ;(client as any).pub.getTransactionReceipt = vi.fn().mockRejectedValue(new Error("not found"))
    ;(client as any).pub.getTransaction = vi.fn().mockResolvedValue({ hash: "0xabc" })
    expect(await client.getTxStatus("0xabc" as `0x${string}`)).toBe("pending")
  })

  it("returns 'unknown' when neither receipt nor tx is found", async () => {
    ;(client as any).pub.getTransactionReceipt = vi.fn().mockRejectedValue(new Error("not found"))
    ;(client as any).pub.getTransaction = vi.fn().mockRejectedValue(new Error("not found"))
    expect(await client.getTxStatus("0xabc" as `0x${string}`)).toBe("unknown")
  })
})

describe("AfiClient — getTokenPrice", () => {
  it("returns {price, inverse} from the underlying quote", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    const fakeQuote = { tokenInPrice: "0.00031", tokenOutPrice: "3225" }
    const builder: any = {
      slippage: vi.fn().mockReturnThis(),
      network:  vi.fn().mockReturnThis(),
      get:      vi.fn().mockResolvedValue(fakeQuote),
    }
    ;(client as any).quote = vi.fn().mockReturnValue(builder)

    const r = await client.getTokenPrice("0xa" as `0x${string}`, "0xb" as `0x${string}`)
    expect(r.price).toBe("0.00031")
    expect(r.inverse).toBe("3225")
  })

  it("respects amount + slippage + network overrides", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    const builder: any = {
      slippage: vi.fn().mockReturnThis(),
      network:  vi.fn().mockReturnThis(),
      get:      vi.fn().mockResolvedValue({ tokenInPrice: "1", tokenOutPrice: "1" }),
    }
    const quoteSpy = vi.fn().mockReturnValue(builder)
    ;(client as any).quote = quoteSpy

    await client.getTokenPrice("0xa" as `0x${string}`, "0xb" as `0x${string}`, {
      amount: "1000", slippage: 1.0, network: "bsc",
    })
    expect(quoteSpy).toHaveBeenCalledWith("0xa", "0xb", "1000")
    expect(builder.slippage).toHaveBeenCalledWith(1.0)
    expect(builder.network).toHaveBeenCalledWith("bsc")
  })
})

describe("AfiClient — preflight", () => {
  const TOKEN = "0xaaaa589fcd6edb6e08f4c7c32d4f71b54bda02913" as const
  const quote: any = {
    tokenIn: TOKEN, tokenOut: "0xbbbb", amountInWei: 1_000_000n, minOutWei: 1n, steps: "0x",
  }

  it("reports INSUFFICIENT_BALANCE when balance < amountIn", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    client.connect(PK)
    // balance then allowance (both via readContract)
    ;(client as any).pub.readContract = vi.fn()
      .mockResolvedValueOnce(100n)      // balance < required
      .mockResolvedValueOnce(0n)        // allowance
    const r = await client.preflight(quote)
    expect(r.canExecute).toBe(false)
    expect(r.problems.some(p => p.code === "INSUFFICIENT_BALANCE")).toBe(true)
    expect(r.needsApproval).toBe(true)
  })

  it("reports needsApproval but canExecute=true when balance ok, allowance missing, no simulate", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    client.connect(PK)
    ;(client as any).pub.readContract = vi.fn()
      .mockResolvedValueOnce(10_000_000n)  // balance ok
      .mockResolvedValueOnce(0n)            // allowance missing
    const r = await client.preflight(quote)
    expect(r.canExecute).toBe(true)
    expect(r.needsApproval).toBe(true)
    expect(r.problems).toHaveLength(0)
  })

  it("runs simulate when balance and allowance are sufficient", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    client.connect(PK)
    ;(client as any).pub.readContract = vi.fn()
      .mockResolvedValueOnce(10_000_000n)
      .mockResolvedValueOnce(10_000_000n)
    const simSpy = vi.fn().mockResolvedValue({})
    ;(client as any).pub.simulateContract = simSpy

    const r = await client.preflight(quote)
    expect(simSpy).toHaveBeenCalled()
    expect(r.canExecute).toBe(true)
    expect(r.needsApproval).toBe(false)
  })

  it("captures SIMULATION_FAILED into problems", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    client.connect(PK)
    ;(client as any).pub.readContract = vi.fn()
      .mockResolvedValueOnce(10_000_000n)
      .mockResolvedValueOnce(10_000_000n)
    ;(client as any).pub.simulateContract = vi.fn().mockRejectedValue({ shortMessage: "minOut" })

    const r = await client.preflight(quote)
    expect(r.canExecute).toBe(false)
    expect(r.problems[0].code).toBe("SIMULATION_FAILED")
    expect(r.problems[0].message).toBe("minOut")
  })
})

describe("AfiClient — nonce management", () => {
  it("getNonce reads the pending nonce from the RPC", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    client.connect(PK)
    ;(client as any).pub.getTransactionCount = vi.fn().mockResolvedValue(42)
    const n = await client.getNonce()
    expect(n).toBe(42)
  })

  it("useManagedNonce starts a local counter", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    client.connect(PK)
    ;(client as any).pub.getTransactionCount = vi.fn().mockResolvedValue(100)
    await client.useManagedNonce()

    const a = (client as any).allocateNonce()
    const b = (client as any).allocateNonce()
    const c = (client as any).allocateNonce(999) // explicit override wins
    expect(a).toBe(100)
    expect(b).toBe(101)
    expect(c).toBe(999)
  })

  it("disableManagedNonce reverts to undefined (let viem fetch)", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    client.connect(PK)
    ;(client as any).pub.getTransactionCount = vi.fn().mockResolvedValue(10)
    await client.useManagedNonce()
    client.disableManagedNonce()
    expect((client as any).allocateNonce()).toBeUndefined()
  })
})
