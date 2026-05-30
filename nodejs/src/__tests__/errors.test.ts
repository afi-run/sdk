import { describe, expect, it } from "vitest"
import {
  AfiError,
  ApprovalError,
  InsufficientBalanceError,
  NoSignerError,
  QuoteError,
  SimulationFailedError,
  SwapRevertedError,
  isAfiError,
  isApprovalError,
  isInsufficientBalanceError,
  isNoSignerError,
  isQuoteError,
  isSimulationFailedError,
  isSwapRevertedError,
} from "../errors.js"

describe("AfiError", () => {
  it("sets message and code", () => {
    const e = new AfiError("test message", "TEST_CODE")
    expect(e.message).toBe("test message")
    expect(e.code).toBe("TEST_CODE")
    expect(e.name).toBe("AfiError")
    expect(e).toBeInstanceOf(Error)
  })
})

describe("InsufficientBalanceError", () => {
  it("includes token, balance and required fields", () => {
    const e = new InsufficientBalanceError("0xabc", 100n, 500n)
    expect(e.token).toBe("0xabc")
    expect(e.balance).toBe(100n)
    expect(e.required).toBe(500n)
    expect(e.code).toBe("INSUFFICIENT_BALANCE")
    expect(e.name).toBe("InsufficientBalanceError")
    expect(e.message).toContain("0xabc")
    expect(e).toBeInstanceOf(AfiError)
  })

  it("uses symbol + formatted decimals when provided", () => {
    const e = new InsufficientBalanceError("0xabc", 500_000n, 1_000_000n, "0xOwner", "USDC", 6)
    expect(e.message).toContain("USDC")
    expect(e.message).toContain("0xOwner")
    expect(e.message).toContain("0.5")  // 500_000 / 10^6
    expect(e.message).toContain("1")    // 1_000_000 / 10^6
    expect(e.symbol).toBe("USDC")
    expect(e.decimals).toBe(6)
    expect(e.owner).toBe("0xOwner")
  })

  it("falls back to raw addresses when symbol/decimals unknown", () => {
    const e = new InsufficientBalanceError("0xabc", 100n, 500n)
    expect(e.message).toMatch(/have 100/)
    expect(e.message).toMatch(/need 500/)
  })
})

describe("QuoteError", () => {
  it("includes reason in message", () => {
    const e = new QuoteError("no route found")
    expect(e.message).toContain("no route found")
    expect(e.code).toBe("QUOTE_FAILED")
    expect(e).toBeInstanceOf(AfiError)
  })
})

describe("SimulationFailedError", () => {
  it("sets reason and optional revertData", () => {
    const e = new SimulationFailedError("slippage exceeded", "0xdeadbeef")
    expect(e.reason).toBe("slippage exceeded")
    expect(e.revertData).toBe("0xdeadbeef")
    expect(e.code).toBe("SIMULATION_FAILED")
    expect(e).toBeInstanceOf(AfiError)
  })

  it("works without revertData", () => {
    const e = new SimulationFailedError("unknown revert")
    expect(e.revertData).toBeUndefined()
  })
})

describe("ApprovalError", () => {
  it("sets code correctly", () => {
    const e = new ApprovalError("allowance mismatch")
    expect(e.code).toBe("APPROVAL_FAILED")
    expect(e.message).toContain("allowance mismatch")
    expect(e).toBeInstanceOf(AfiError)
  })
})

describe("SwapRevertedError", () => {
  it("sets reason field", () => {
    const e = new SwapRevertedError("minOut not met")
    expect(e.reason).toBe("minOut not met")
    expect(e.code).toBe("SWAP_REVERTED")
    expect(e).toBeInstanceOf(AfiError)
  })
})

describe("NoSignerError", () => {
  it("sets NO_SIGNER code and is an AfiError", () => {
    const e = new NoSignerError()
    expect(e.code).toBe("NO_SIGNER")
    expect(e.name).toBe("NoSignerError")
    expect(e.message).toContain("Private key")
    expect(e).toBeInstanceOf(AfiError)
  })
})

describe("type guards", () => {
  it("isAfiError narrows AfiError-shaped errors", () => {
    expect(isAfiError(new QuoteError("x"))).toBe(true)
    expect(isAfiError(new Error("plain"))).toBe(false)
    expect(isAfiError("string")).toBe(false)
    expect(isAfiError(null)).toBe(false)
    expect(isAfiError(undefined)).toBe(false)
  })

  it("each guard matches only its own code", () => {
    const cases: Array<[Function, Error]> = [
      [isInsufficientBalanceError, new InsufficientBalanceError("0xt", 1n, 2n)],
      [isQuoteError,               new QuoteError("x")],
      [isSimulationFailedError,    new SimulationFailedError("x")],
      [isApprovalError,            new ApprovalError("x")],
      [isSwapRevertedError,        new SwapRevertedError("x")],
      [isNoSignerError,            new NoSignerError()],
    ]
    for (const [guard, owner] of cases) {
      expect((guard as any)(owner)).toBe(true)
      for (const [otherGuard, other] of cases) {
        if (other === owner) continue
        expect((otherGuard as any)(owner)).toBe(false)
      }
    }
  })

  it("works without instanceof — guards inspect .code, not the prototype chain", () => {
    const fake = Object.assign(new Error("simulated revert"), { code: "SIMULATION_FAILED" })
    expect(isSimulationFailedError(fake)).toBe(true)
    expect(isAfiError(fake)).toBe(true)
  })
})
