import { describe, expect, it } from "vitest"
import { lookupBalanceSlot, registerBalanceSlot } from "../token-slots.js"
import type { Address } from "../types.js"

const USDC_BASE: Address = "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
const UNKNOWN: Address   = "0x9999999999999999999999999999999999999999"

describe("lookupBalanceSlot", () => {
  it("returns undefined for a chain with no known slots", () => {
    expect(lookupBalanceSlot(999999, USDC_BASE)).toBeUndefined()
  })

  it("returns undefined for an unknown token on a known chain", () => {
    expect(lookupBalanceSlot(8453, UNKNOWN)).toBeUndefined()
  })

  it("is case-insensitive once a slot is registered", () => {
    registerBalanceSlot(8453, USDC_BASE, 9)
    expect(lookupBalanceSlot(8453, USDC_BASE.toLowerCase() as Address)).toBe(9)
    expect(lookupBalanceSlot(8453, USDC_BASE.toUpperCase().replace("0X", "0x") as Address)).toBe(9)
  })
})

describe("registerBalanceSlot", () => {
  it("creates the chain map on first registration and reuses it afterwards", () => {
    const NEW_CHAIN = 424242
    expect(lookupBalanceSlot(NEW_CHAIN, USDC_BASE)).toBeUndefined()

    // First call: chain map does not exist yet → it is created.
    registerBalanceSlot(NEW_CHAIN, USDC_BASE, 3)
    expect(lookupBalanceSlot(NEW_CHAIN, USDC_BASE)).toBe(3)

    // Second call on the same chain: map already exists → reused branch.
    registerBalanceSlot(NEW_CHAIN, UNKNOWN, 7)
    expect(lookupBalanceSlot(NEW_CHAIN, USDC_BASE)).toBe(3)
    expect(lookupBalanceSlot(NEW_CHAIN, UNKNOWN)).toBe(7)
  })
})
