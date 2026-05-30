import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { encodeEventTopics, encodeAbiParameters } from "viem"
import { AFI_ABI, AFI_ADDRESS } from "../constants.js"
import { applyGasBuffer, parseSwapResult, simulateSwap, submitSwap } from "../swap.js"
import { SimulationFailedError, SwapRevertedError } from "../errors.js"
import type { Account, PublicClient, Transport, WalletClient } from "viem"
import type { base } from "viem/chains"
import type { Quote } from "../types.js"

type MockPublic = Partial<PublicClient<Transport, typeof base>> & {
  simulateContract: ReturnType<typeof vi.fn>
  estimateContractGas: ReturnType<typeof vi.fn>
  waitForTransactionReceipt: ReturnType<typeof vi.fn>
}

type MockWallet = Partial<WalletClient<Transport, typeof base, Account>> & {
  writeContract: ReturnType<typeof vi.fn>
}

const SENDER = "0x1234567890123456789012345678901234567890" as const

const fakeQuote: Quote = {
  tokenIn:      "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
  tokenOut:     "0x4200000000000000000000000000000000000006",
  amountIn:     "100",
  amountOut:    "0.04",
  minOut:       "0.039",
  amountInWei:  100_000000n,
  amountOutWei: 40_000_000_000_000_000n,
  minOutWei:    39_000_000_000_000_000n,
  steps:        "0x" as `0x${string}`,
  path:         [],
  hops:         [],
  slippage:     0.5,
  feeBps:       35,
  tokenInPrice: "1",
  tokenOutPrice: "1",
  createdAt:     Date.now(),
  network:       "base",
  maxHops:       2,
}

describe("applyGasBuffer", () => {
  it("adds the percentage on top of estimated gas", () => {
    expect(applyGasBuffer(100n, 15)).toBe(115n)
    expect(applyGasBuffer(1000n, 25)).toBe(1250n)
    expect(applyGasBuffer(100n, 0)).toBe(100n)
  })

  it("returns gas unchanged when buffer is negative", () => {
    expect(applyGasBuffer(100n, -5)).toBe(100n)
  })

  it("floors fractional buffer", () => {
    expect(applyGasBuffer(100n, 15.9)).toBe(115n)
  })
})

describe("simulateSwap", () => {
  let pub: MockPublic

  beforeEach(() => {
    pub = {
      simulateContract:          vi.fn(),
      estimateContractGas:       vi.fn(),
      waitForTransactionReceipt: vi.fn(),
    }
  })

  afterEach(() => { vi.clearAllMocks() })

  it("resolves silently when simulation succeeds", async () => {
    pub.simulateContract.mockResolvedValue({})
    await expect(simulateSwap(fakeQuote, SENDER, pub as any)).resolves.toBeUndefined()
  })

  it("throws SimulationFailedError carrying the revert reason", async () => {
    pub.simulateContract.mockRejectedValue({ shortMessage: "minOut not met", data: "0xdead" })
    try {
      await simulateSwap(fakeQuote, SENDER, pub as any)
      throw new Error("should not reach here")
    } catch (e) {
      expect(e).toBeInstanceOf(SimulationFailedError)
      expect((e as SimulationFailedError).reason).toBe("minOut not met")
      expect((e as SimulationFailedError).revertData).toBe("0xdead")
    }
  })

  it("falls back to message when shortMessage is missing", async () => {
    pub.simulateContract.mockRejectedValue({ message: "execution reverted" })
    await expect(simulateSwap(fakeQuote, SENDER, pub as any))
      .rejects.toBeInstanceOf(SimulationFailedError)
  })
})

describe("submitSwap", () => {
  let pub: MockPublic
  let wallet: MockWallet

  beforeEach(() => {
    pub = {
      simulateContract:          vi.fn(),
      estimateContractGas:       vi.fn().mockResolvedValue(200_000n),
      waitForTransactionReceipt: vi.fn(),
    }
    wallet = { writeContract: vi.fn().mockResolvedValue("0xabc") }
  })

  afterEach(() => { vi.clearAllMocks() })

  it("passes gas estimate + buffer to writeContract", async () => {
    await submitSwap(fakeQuote, SENDER, pub as any, wallet as any, 15)
    expect(wallet.writeContract).toHaveBeenCalledWith(
      expect.objectContaining({ gas: 230_000n }), // 200k * 1.15
    )
  })

  it("uses 0 buffer when configured", async () => {
    await submitSwap(fakeQuote, SENDER, pub as any, wallet as any, 0)
    expect(wallet.writeContract).toHaveBeenCalledWith(
      expect.objectContaining({ gas: 200_000n }),
    )
  })

  it("surfaces revert reason when estimateGas fails with a known revert", async () => {
    pub.estimateContractGas.mockRejectedValue(new Error("execution reverted"))
    pub.simulateContract.mockRejectedValue({ shortMessage: "InsufficientAllowance" })

    await expect(submitSwap(fakeQuote, SENDER, pub as any, wallet as any, 15))
      .rejects.toThrow(/InsufficientAllowance/)
  })

  it("wraps writeContract errors as SwapRevertedError", async () => {
    wallet.writeContract.mockRejectedValue(new Error("user rejected"))
    await expect(submitSwap(fakeQuote, SENDER, pub as any, wallet as any, 15))
      .rejects.toBeInstanceOf(SwapRevertedError)
  })
})

describe("parseSwapResult", () => {
  const FROM = "0x1234567890123456789012345678901234567890" as const
  const ASSET_IN = "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913" as const
  const ASSET_OUT = "0x4200000000000000000000000000000000000006" as const

  function makeLog() {
    const topics = encodeEventTopics({
      abi: AFI_ABI,
      eventName: "SwapExecuted",
      args: { from: FROM, assetIn: ASSET_IN, assetOut: ASSET_OUT },
    }) as `0x${string}`[]
    const data = encodeAbiParameters(
      [{ type: "uint256" }, { type: "uint256" }],
      [1_000_000n, 500_000_000_000_000_000n],
    )
    return { address: AFI_ADDRESS, topics, data }
  }

  it("returns null when no SwapExecuted log is present", () => {
    const result = parseSwapResult({ logs: [], transactionHash: "0xabc", blockNumber: 1n, gasUsed: 0n } as any)
    expect(result).toBeNull()
  })

  it("decodes the SwapExecuted event into a SwapResult", () => {
    const log = makeLog()
    const result = parseSwapResult({
      logs: [log],
      transactionHash: "0xabc" as `0x${string}`,
      blockNumber: 42n,
      gasUsed: 150_000n,
    } as any)

    expect(result).not.toBeNull()
    expect(result!.txHash).toBe("0xabc")
    expect(result!.blockNumber).toBe(42n)
    expect(result!.amountIn).toBe(1_000_000n)
    expect(result!.amountOut).toBe(500_000_000_000_000_000n)
    expect(result!.tokenIn.toLowerCase()).toBe(ASSET_IN)
    expect(result!.tokenOut.toLowerCase()).toBe(ASSET_OUT)
    expect(result!.gasUsed).toBe(150_000n)
  })
})
