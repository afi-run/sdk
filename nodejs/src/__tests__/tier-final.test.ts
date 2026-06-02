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

    const a = await (client as any).allocateNonce()
    const b = await (client as any).allocateNonce()
    const c = await (client as any).allocateNonce(999) // explicit override wins
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
    await expect((client as any).allocateNonce()).resolves.toBeUndefined()
  })

  it("serializes parallel allocations via a Promise queue (mutex)", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    client.connect(PK)
    // Simulate a slow getTransactionCount that resolves only after a tick —
    // without the mutex, multiple callers would race past the fetch and reuse
    // the same starting nonce.
    let delayed: (() => void) | undefined
    ;(client as any).pub.getTransactionCount = vi.fn(
      () =>
        new Promise<number>((resolve) => {
          delayed = () => resolve(50)
        }),
    )
    // Trigger managed mode without pre-seeding the local counter, so the very
    // first allocation has to fetch.
    ;(client as any)._managedNonce = true
    ;(client as any)._localNonce = null

    const promises = Array.from({ length: 5 }, () => (client as any).allocateNonce() as Promise<number>)
    // Yield so the first allocateNonce call actually invokes getTransactionCount.
    await new Promise<void>((r) => setImmediate(r))
    delayed!()
    const nonces = await Promise.all(promises)
    expect(nonces).toEqual([50, 51, 52, 53, 54])
    // Only one fetch despite five parallel callers.
    expect((client as any).pub.getTransactionCount).toHaveBeenCalledTimes(1)
  })
})

describe("AfiClient — sendContractTx revert capture", () => {
  it("falls back to pub.call to surface the revert reason", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    client.connect(PK)

    ;(client as any).pub.estimateContractGas = vi.fn().mockRejectedValue(new Error("opaque RPC error"))
    const callErr = Object.assign(new Error("execution reverted: insufficient liquidity"), {
      shortMessage: "execution reverted: insufficient liquidity",
    })
    ;(client as any).pub.call = vi.fn().mockRejectedValue(callErr)

    await expect(
      client.sendContractTx(
        "0x1111111111111111111111111111111111111111" as any,
        [{ type: "function", name: "pause", inputs: [], outputs: [], stateMutability: "nonpayable" }] as const,
        "pause",
        [],
      ),
    ).rejects.toThrow(/pause would revert: execution reverted: insufficient liquidity/)
  })

  it("rethrows original estimate error when pub.call does not revert", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    client.connect(PK)

    ;(client as any).pub.estimateContractGas = vi.fn().mockRejectedValue(new Error("flaky estimate"))
    ;(client as any).pub.call = vi.fn().mockResolvedValue({ data: "0x" })

    await expect(
      client.sendContractTx(
        "0x1111111111111111111111111111111111111111" as any,
        [{ type: "function", name: "pause", inputs: [], outputs: [], stateMutability: "nonpayable" }] as const,
        "pause",
        [],
      ),
    ).rejects.toThrow(/estimateContractGas\(pause\): flaky estimate/)
  })
})

describe("AfiClient — swapFor precheck", () => {
  const USER  = "0x2222222222222222222222222222222222222222" as const
  const TOKEN = "0x3333333333333333333333333333333333333333" as const

  it("throws ApprovalError when allowance is below amountIn", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    client.connect(PK)
    ;(client as any).pub.readContract = vi.fn().mockResolvedValue(100n) // allowance
    await expect(
      client.swapFor({
        user: USER,
        tokenIn: TOKEN,
        tokenOut: TOKEN,
        amountIn: 1_000n,
        minOut: 0n,
        steps: "0x",
      }),
    ).rejects.toMatchObject({ code: "APPROVAL_FAILED" })
  })

  it("proceeds when allowance is sufficient", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    client.connect(PK)
    ;(client as any).pub.readContract = vi.fn().mockResolvedValue(10_000n)
    ;(client as any).pub.estimateContractGas = vi.fn().mockResolvedValue(50_000n)
    const writeSpy = vi.fn().mockResolvedValue("0xswapfor")
    ;(client as any).wallet = { writeContract: writeSpy }
    ;(client as any).pub.waitForTransactionReceipt = vi.fn().mockResolvedValue({ blockNumber: 1n, gasUsed: 0n })

    await client.swapFor({
      user: USER,
      tokenIn: TOKEN,
      tokenOut: TOKEN,
      amountIn: 1_000n,
      minOut: 0n,
      steps: "0x",
    })
    expect(writeSpy).toHaveBeenCalledTimes(1)
  })

  it("skips allowance read when precheck=false", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    client.connect(PK)
    const readSpy = vi.fn().mockResolvedValue(0n)
    ;(client as any).pub.readContract = readSpy
    ;(client as any).pub.estimateContractGas = vi.fn().mockResolvedValue(50_000n)
    ;(client as any).wallet = { writeContract: vi.fn().mockResolvedValue("0xswapfor") }
    ;(client as any).pub.waitForTransactionReceipt = vi.fn().mockResolvedValue({ blockNumber: 1n, gasUsed: 0n })

    await client.swapFor({
      user: USER,
      tokenIn: TOKEN,
      tokenOut: TOKEN,
      amountIn: 1_000n,
      minOut: 0n,
      steps: "0x",
      precheck: false,
    })
    expect(readSpy).not.toHaveBeenCalled()
  })
})

describe("AfiClient — acceptOwnership", () => {
  const C1 = "0x1111111111111111111111111111111111111111" as const
  const C2 = "0x2222222222222222222222222222222222222222" as const

  it("encodes acceptOwnership and forwards to sendContractTx", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    client.connect(PK)
    ;(client as any).pub.estimateContractGas = vi.fn().mockResolvedValue(40_000n)
    const writeSpy = vi.fn().mockResolvedValue("0xaccept")
    ;(client as any).wallet = { writeContract: writeSpy }
    ;(client as any).pub.waitForTransactionReceipt = vi.fn().mockResolvedValue({ blockNumber: 1n, gasUsed: 0n })

    await client.acceptOwnership(C1)
    expect(writeSpy).toHaveBeenCalledTimes(1)
    expect(writeSpy.mock.calls[0][0].functionName).toBe("acceptOwnership")
    expect(writeSpy.mock.calls[0][0].address).toBe(C1)
  })

  it("acceptOwnershipBatch processes contracts sequentially", async () => {
    const client = new AfiClient({ rpcUrl: "http://localhost:1" })
    client.connect(PK)
    ;(client as any).pub.estimateContractGas = vi.fn().mockResolvedValue(40_000n)
    const writeSpy = vi.fn().mockResolvedValue("0xaccept")
    ;(client as any).wallet = { writeContract: writeSpy }
    ;(client as any).pub.waitForTransactionReceipt = vi.fn().mockResolvedValue({ blockNumber: 1n, gasUsed: 0n })

    const receipts = await client.acceptOwnershipBatch([C1, C2])
    expect(receipts).toHaveLength(2)
    expect(writeSpy).toHaveBeenCalledTimes(2)
    expect(writeSpy.mock.calls[0][0].address).toBe(C1)
    expect(writeSpy.mock.calls[1][0].address).toBe(C2)
  })
})
