import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { AFI_ADDRESS } from "../constants.js"
import { InsufficientBalanceError, ApprovalError } from "../errors.js"
import {
  assertSufficientBalance,
  ensureExactApproval,
  getAllowance,
  getBalance,
  getDecimals,
} from "../token.js"
import type { Account, PublicClient, Transport, WalletClient } from "viem"
import type { base } from "viem/chains"

type MockPublicClient = Partial<PublicClient<Transport, typeof base>> & {
  readContract: ReturnType<typeof vi.fn>
  waitForTransactionReceipt: ReturnType<typeof vi.fn>
}

type MockWalletClient = Partial<WalletClient<Transport, typeof base, Account>> & {
  writeContract: ReturnType<typeof vi.fn>
}

const TOKEN = "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913" as const
const OWNER = "0x1234567890123456789012345678901234567890" as const

let pub: MockPublicClient
let wallet: MockWalletClient

beforeEach(() => {
  pub = {
    readContract: vi.fn(),
    waitForTransactionReceipt: vi.fn().mockResolvedValue({}),
  }
  wallet = {
    writeContract: vi.fn().mockResolvedValue("0xapprovetxhash"),
  }
})

afterEach(() => {
  vi.clearAllMocks()
})

describe("getDecimals", () => {
  it("calls readContract with decimals", async () => {
    pub.readContract!.mockResolvedValue(6)
    const result = await getDecimals(TOKEN, pub as any)
    expect(result).toBe(6)
    expect(pub.readContract).toHaveBeenCalledWith(
      expect.objectContaining({ functionName: "decimals", address: TOKEN }),
    )
  })
})

describe("getBalance", () => {
  it("returns balance from readContract", async () => {
    pub.readContract!.mockResolvedValue(500_000000n)
    const result = await getBalance(TOKEN, OWNER, pub as any)
    expect(result).toBe(500_000000n)
    expect(pub.readContract).toHaveBeenCalledWith(
      expect.objectContaining({ functionName: "balanceOf", args: [OWNER] }),
    )
  })
})

describe("getAllowance", () => {
  it("queries allowance for AFI_ADDRESS as spender", async () => {
    pub.readContract!.mockResolvedValue(1000_000000n)
    const result = await getAllowance(TOKEN, OWNER, pub as any)
    expect(result).toBe(1000_000000n)
    expect(pub.readContract).toHaveBeenCalledWith(
      expect.objectContaining({
        functionName: "allowance",
        args: [OWNER, AFI_ADDRESS],
      }),
    )
  })
})

describe("assertSufficientBalance", () => {
  it("passes when balance >= required", async () => {
    pub.readContract!.mockResolvedValue(1000_000000n)
    await expect(
      assertSufficientBalance(TOKEN, OWNER, 500_000000n, pub as any),
    ).resolves.toBeUndefined()
  })

  it("throws InsufficientBalanceError when balance < required", async () => {
    pub.readContract!.mockResolvedValue(100_000000n)
    await expect(
      assertSufficientBalance(TOKEN, OWNER, 500_000000n, pub as any),
    ).rejects.toBeInstanceOf(InsufficientBalanceError)
  })

  it("includes balance and required in error", async () => {
    pub.readContract!.mockResolvedValue(100_000000n)
    try {
      await assertSufficientBalance(TOKEN, OWNER, 500_000000n, pub as any)
    } catch (e) {
      const err = e as InsufficientBalanceError
      expect(err.balance).toBe(100_000000n)
      expect(err.required).toBe(500_000000n)
      expect(err.token).toBe(TOKEN)
    }
  })
})

describe("ensureExactApproval", () => {
  it("skips approval when allowance is already sufficient", async () => {
    pub.readContract!.mockResolvedValue(1000_000000n) // allowance >= amount
    const result = await ensureExactApproval(TOKEN, OWNER, 500_000000n, pub as any, wallet as any)
    expect(result).toBeNull()
    expect(wallet.writeContract).not.toHaveBeenCalled()
  })

  it("sends approve when allowance is insufficient", async () => {
    pub.readContract!
      .mockResolvedValueOnce(0n)          // allowance (before approve)
      .mockResolvedValueOnce(500_000000n) // allowance (after approve - re-check)
    wallet.writeContract!.mockResolvedValue("0xapprovetx")

    const result = await ensureExactApproval(TOKEN, OWNER, 500_000000n, pub as any, wallet as any)

    expect(result).toBe("0xapprovetx")
    expect(wallet.writeContract).toHaveBeenCalledWith(
      expect.objectContaining({
        functionName: "approve",
        args: [AFI_ADDRESS, 500_000000n],
      }),
    )
  })

  it("resets allowance to 0 before re-approving for USDT-style tokens", async () => {
    pub.readContract!
      .mockResolvedValueOnce(200_000000n)  // current allowance > 0
      .mockResolvedValueOnce(500_000000n)  // after approve re-check
    wallet.writeContract!.mockResolvedValue("0xtxhash")

    await ensureExactApproval(TOKEN, OWNER, 500_000000n, pub as any, wallet as any)

    // First call should be reset to 0
    expect(wallet.writeContract).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({ functionName: "approve", args: [AFI_ADDRESS, 0n] }),
    )
    // Second call should be the actual approve
    expect(wallet.writeContract).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({ functionName: "approve", args: [AFI_ADDRESS, 500_000000n] }),
    )
  })

  it("continues when reset-to-zero throws (token does not require reset)", async () => {
    pub.readContract!
      .mockResolvedValueOnce(200_000000n)  // current allowance > 0
      .mockResolvedValueOnce(500_000000n)  // after approve re-check
    wallet.writeContract!
      .mockRejectedValueOnce(new Error("reset not supported")) // reset fails
      .mockResolvedValueOnce("0xtxhash")                      // actual approve succeeds

    const result = await ensureExactApproval(TOKEN, OWNER, 500_000000n, pub as any, wallet as any)
    expect(result).toBe("0xtxhash")
  })

  it("throws ApprovalError when on-chain allowance does not match after confirmation", async () => {
    pub.readContract!
      .mockResolvedValueOnce(0n)   // initial allowance
      .mockResolvedValueOnce(0n)   // post-approve re-check still 0 (failure scenario)
    wallet.writeContract!.mockResolvedValue("0xtxhash")

    await expect(
      ensureExactApproval(TOKEN, OWNER, 500_000000n, pub as any, wallet as any),
    ).rejects.toBeInstanceOf(ApprovalError)
  })

  it("throws ApprovalError when writeContract fails", async () => {
    pub.readContract!.mockResolvedValue(0n)
    wallet.writeContract!.mockRejectedValue(new Error("user rejected"))

    await expect(
      ensureExactApproval(TOKEN, OWNER, 500_000000n, pub as any, wallet as any),
    ).rejects.toBeInstanceOf(ApprovalError)
  })
})
