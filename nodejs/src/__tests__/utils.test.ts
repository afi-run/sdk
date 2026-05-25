import { describe, expect, it } from "vitest"
import { formatAmount, formatUnits, parseUnits } from "../utils.js"

describe("formatAmount", () => {
  it("converts whole USDC amount (6 decimals)", () => {
    expect(formatAmount(1000_000000n, 6)).toBe("1000")
  })

  it("converts fractional USDC amount", () => {
    expect(formatAmount(1_500000n, 6)).toBe("1.5")
  })

  it("trims trailing zeros in fractional part", () => {
    expect(formatAmount(1_100000n, 6)).toBe("1.1")
  })

  it("handles sub-unit amounts (less than 1 token)", () => {
    expect(formatAmount(123456n, 6)).toBe("0.123456")
  })

  it("handles WETH with 18 decimals", () => {
    expect(formatAmount(1_000000000000000000n, 18)).toBe("1")
  })

  it("handles fractional WETH", () => {
    expect(formatAmount(500000000000000000n, 18)).toBe("0.5")
  })

  it("handles zero", () => {
    expect(formatAmount(0n, 6)).toBe("0")
  })

  it("handles very large amount", () => {
    expect(formatAmount(1000000_000000n, 6)).toBe("1000000")
  })

  it("preserves all significant decimal digits", () => {
    expect(formatAmount(1_000001n, 6)).toBe("1.000001")
  })
})

describe("formatUnits", () => {
  it("is an alias for formatAmount", () => {
    expect(formatUnits(1000_000000n, 6)).toBe("1000")
    expect(formatUnits(1_500000n, 6)).toBe("1.5")
    expect(formatUnits(0n, 6)).toBe("0")
  })
})

describe("parseUnits", () => {
  it("parses whole USDC amount", () => {
    expect(parseUnits("1000", 6)).toBe(1000_000000n)
  })

  it("parses fractional USDC amount", () => {
    expect(parseUnits("1.5", 6)).toBe(1_500000n)
  })

  it("parses sub-unit amount", () => {
    expect(parseUnits("0.123456", 6)).toBe(123456n)
  })

  it("parses whole WETH (18 decimals)", () => {
    expect(parseUnits("1", 18)).toBe(1_000000000000000000n)
  })

  it("parses fractional WETH", () => {
    expect(parseUnits("0.5", 18)).toBe(500000000000000000n)
  })

  it("pads missing decimal places with zeros", () => {
    expect(parseUnits("1.1", 6)).toBe(1_100000n)
  })

  it("parses zero", () => {
    expect(parseUnits("0", 6)).toBe(0n)
  })

  it("roundtrips with formatUnits", () => {
    const raw = 1_234500n
    expect(parseUnits(formatUnits(raw, 6), 6)).toBe(raw)
  })

  it("handles zero decimals (whole number only)", () => {
    expect(parseUnits("5", 0)).toBe(5n)
  })
})
