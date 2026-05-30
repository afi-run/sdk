import { describe, expect, it } from "vitest"
import { applySlippage, calculateMinOut } from "../slippage.js"

describe("applySlippage", () => {
  it("returns amount * (1 - slippagePct/100)", () => {
    expect(applySlippage(10_000n, 0.5)).toBe(9_950n)   // 0.5% off
    expect(applySlippage(10_000n, 1.0)).toBe(9_900n)   // 1.0% off
    expect(applySlippage(10_000n, 5.0)).toBe(9_500n)   // 5.0% off
  })

  it("handles fractional slippage with bps rounding", () => {
    expect(applySlippage(10_000n, 1.25)).toBe(9_875n)  // 125 bps
    expect(applySlippage(10_000n, 0.01)).toBe(9_999n)  // 1 bps
  })

  it("0% returns the input unchanged", () => {
    expect(applySlippage(12345n, 0)).toBe(12345n)
  })

  it("negative slippage is clamped to 0", () => {
    expect(applySlippage(10_000n, -1)).toBe(10_000n)
  })

  it(">= 100% returns 0", () => {
    expect(applySlippage(10_000n, 100)).toBe(0n)
    expect(applySlippage(10_000n, 150)).toBe(0n)
  })

  it("works for large wei amounts", () => {
    const oneEther = 10n ** 18n
    expect(applySlippage(oneEther, 0.5)).toBe(995_000_000_000_000_000n) // 0.995 ETH
  })
})

describe("calculateMinOut", () => {
  it("is an alias of applySlippage", () => {
    expect(calculateMinOut(10_000n, 0.5)).toBe(applySlippage(10_000n, 0.5))
  })
})
