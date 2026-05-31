import { describe, expect, it } from "vitest"
import { keccak256, encodeAbiParameters, getAddress } from "viem"
import { mappingSlot, simulateRoute } from "../simulate.js"
import { lookupBalanceSlot, registerBalanceSlot } from "../token-slots.js"
import type { Address } from "../types.js"

describe("lookupBalanceSlot", () => {
  it("returns slot 3 for Base WETH", () => {
    const slot = lookupBalanceSlot(8453, "0x4200000000000000000000000000000000000006")
    expect(slot).toBe(3)
  })

  it("is case-insensitive", () => {
    const slot = lookupBalanceSlot(8453, "0x4200000000000000000000000000000000000006".toUpperCase() as Address)
    expect(slot).toBe(3)
  })

  it("returns undefined for unknown token", () => {
    const slot = lookupBalanceSlot(8453, "0x000000000000000000000000000000000000dEaD")
    expect(slot).toBeUndefined()
  })
})

describe("registerBalanceSlot", () => {
  it("adds runtime entries readable by lookup", () => {
    const addr = "0x000000000000000000000000000000000000bEEF" as Address
    registerBalanceSlot(8453, addr, 42)
    expect(lookupBalanceSlot(8453, addr)).toBe(42)
  })
})

describe("mappingSlot", () => {
  it("matches keccak256(holder . slot) by hand", () => {
    const holder = getAddress("0x1234567890abcdef1234567890abcdef12345678")
    const slot = 9
    const expected = keccak256(
      encodeAbiParameters(
        [{ type: "address" }, { type: "uint256" }],
        [holder, BigInt(slot)],
      ),
    )
    expect(mappingSlot(holder, slot)).toBe(expected)
  })

  it("is deterministic", () => {
    const h = getAddress("0x1234567890123456789012345678901234567890")
    expect(mappingSlot(h, 5)).toBe(mappingSlot(h, 5))
  })

  it("varies with slot", () => {
    const h = getAddress("0x1234567890123456789012345678901234567890")
    expect(mappingSlot(h, 1)).not.toBe(mappingSlot(h, 2))
  })
})

describe("simulateRoute input validation", () => {
  it("rejects zero amount", async () => {
    await expect(
      simulateRoute({
        publicClient: {} as never,
        chainId: 8453,
        quoterAddress: "0x0000000000000000000000000000000000000001",
        asset: "0x4200000000000000000000000000000000000006",
        amount: 0n,
        stepsEncoded: "0x00",
      }),
    ).rejects.toThrow(/amount must be > 0/)
  })

  it("rejects unknown token", async () => {
    await expect(
      simulateRoute({
        publicClient: {} as never,
        chainId: 8453,
        quoterAddress: "0x0000000000000000000000000000000000000001",
        asset: "0x000000000000000000000000000000000000cAfE",
        amount: 1n,
        stepsEncoded: "0x00",
      }),
    ).rejects.toThrow(/no static balance slot/)
  })
})
