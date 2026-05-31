import { describe, expect, it } from "vitest"
import { encodeSteps, type Step } from "../encode-steps.js"

describe("encodeSteps", () => {
  it("returns just the u8 count for an empty step list", () => {
    expect(encodeSteps([])).toBe("0x00")
  })

  it("packs a single step in tight format", () => {
    const out = encodeSteps([{ id: 3, data: "0x0102" }])
    // 01 (numSteps) | 0003 (id) | 0002 (dataLen) | 0102 (data)
    expect(out).toBe("0x0100030002" + "0102")
  })

  it("packs multiple steps in order", () => {
    const out = encodeSteps([
      { id: 1, data: "0xaa" },
      { id: 0xffff, data: "0x" },
    ])
    // 02 | 0001 0001 aa | ffff 0000
    expect(out).toBe("0x020001" + "0001aa" + "ffff0000")
  })

  it("throws on more than 255 steps", () => {
    const steps: Step[] = Array.from({ length: 256 }, () => ({ id: 0, data: "0x" }))
    expect(() => encodeSteps(steps)).toThrow(/exceeds u8 max/)
  })

  it("throws on data exceeding 65535 bytes", () => {
    const big = ("0x" + "ab".repeat(65_536)) as `0x${string}`
    expect(() => encodeSteps([{ id: 0, data: big }])).toThrow(/exceeds u16 max/)
  })
})
