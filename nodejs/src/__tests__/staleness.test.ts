import { describe, expect, it } from "vitest"
import { isQuoteStale } from "../types.js"
import type { Quote } from "../types.js"

function fakeQuote(createdAt: number): Quote {
  return { createdAt } as unknown as Quote
}

describe("isQuoteStale", () => {
  it("returns false for a fresh quote", () => {
    expect(isQuoteStale(fakeQuote(Date.now()), 60)).toBe(false)
  })

  it("returns true when older than maxAgeSec", () => {
    const stale = fakeQuote(Date.now() - 120_000) // 120s old
    expect(isQuoteStale(stale, 60)).toBe(true)
  })

  it("returns false when exactly within maxAgeSec", () => {
    const borderline = fakeQuote(Date.now() - 30_000) // 30s old
    expect(isQuoteStale(borderline, 60)).toBe(false)
  })

  it("returns false when createdAt is 0 / missing (backward-compatible)", () => {
    expect(isQuoteStale(fakeQuote(0), 1)).toBe(false)
  })
})
