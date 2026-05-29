import { describe, expect, it } from "vitest"
import {
  AfiError,
  ApprovalError,
  InsufficientBalanceError,
  NoSignerError,
  QuoteError,
  SimulationFailedError,
  SwapRevertedError,
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
