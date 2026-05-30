import { describe, expect, it } from "vitest"
import { addressUrl, txUrl } from "../explorer.js"
import { NETWORK } from "../types.js"

const HASH = "0xabcdef1234"
const ADDR = "0xdeadbeef0000000000000000000000000000abcd"

describe("txUrl", () => {
  it("defaults to Base explorer", () => {
    expect(txUrl(HASH)).toBe(`https://basescan.org/tx/${HASH}`)
  })

  it("respects the network argument", () => {
    expect(txUrl(HASH, NETWORK.BSC)).toBe(`https://bscscan.com/tx/${HASH}`)
    expect(txUrl(HASH, NETWORK.ARBITRUM)).toBe(`https://arbiscan.io/tx/${HASH}`)
    expect(txUrl(HASH, NETWORK.ETHEREUM)).toBe(`https://etherscan.io/tx/${HASH}`)
    expect(txUrl(HASH, NETWORK.UNICHAIN)).toBe(`https://uniscan.xyz/tx/${HASH}`)
  })

  it("accepts a custom explorer base", () => {
    expect(txUrl(HASH, NETWORK.BASE, "https://my.explorer")).toBe(`https://my.explorer/tx/${HASH}`)
  })

  it("strips trailing slashes from the explorer base", () => {
    expect(txUrl(HASH, NETWORK.BASE, "https://my.explorer///")).toBe(`https://my.explorer/tx/${HASH}`)
  })

  it("throws on unknown network", () => {
    expect(() => txUrl(HASH, "unknown" as any)).toThrow(/no explorer URL/)
  })
})

describe("addressUrl", () => {
  it("returns the address path", () => {
    expect(addressUrl(ADDR)).toBe(`https://basescan.org/address/${ADDR}`)
  })

  it("respects network override", () => {
    expect(addressUrl(ADDR, NETWORK.BSC)).toBe(`https://bscscan.com/address/${ADDR}`)
  })
})
