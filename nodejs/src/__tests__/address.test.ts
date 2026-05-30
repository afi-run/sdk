import { describe, expect, it } from "vitest"
import {
  ZERO_ADDRESS,
  checksumAddress,
  equalAddresses,
  isAddress,
  isZeroAddress,
} from "../address.js"

describe("isAddress", () => {
  it("accepts valid hex addresses", () => {
    expect(isAddress("0xB8cC65321d169D55b93b4402D795701c6B308ce4")).toBe(true)
    expect(isAddress("0x0000000000000000000000000000000000000000")).toBe(true)
  })

  it("rejects invalid inputs", () => {
    expect(isAddress("0xshort")).toBe(false)
    expect(isAddress("notanaddress")).toBe(false)
    expect(isAddress("B8cC65321d169D55b93b4402D795701c6B308ce4")).toBe(false) // no 0x
    expect(isAddress("0xZ8cC65321d169D55b93b4402D795701c6B308ce4")).toBe(false) // non-hex char
  })
})

describe("checksumAddress", () => {
  it("returns EIP-55 checksum form", () => {
    expect(checksumAddress("0xb8cc65321d169d55b93b4402d795701c6b308ce4"))
      .toBe("0xB8cC65321d169D55b93b4402D795701c6B308ce4")
  })

  it("is idempotent for already-checksummed input", () => {
    const checked = checksumAddress("0xB8cC65321d169D55b93b4402D795701c6B308ce4")
    expect(checksumAddress(checked)).toBe(checked)
  })
})

describe("isZeroAddress", () => {
  it("matches the canonical zero address regardless of case", () => {
    expect(isZeroAddress(ZERO_ADDRESS)).toBe(true)
    expect(isZeroAddress("0x0000000000000000000000000000000000000000")).toBe(true)
    expect(isZeroAddress("0X0000000000000000000000000000000000000000")).toBe(true)
  })

  it("returns false for non-zero addresses", () => {
    expect(isZeroAddress("0xB8cC65321d169D55b93b4402D795701c6B308ce4")).toBe(false)
  })
})

describe("equalAddresses", () => {
  it("matches regardless of case", () => {
    expect(equalAddresses(
      "0xB8cC65321d169D55b93b4402D795701c6B308ce4",
      "0xb8cc65321d169d55b93b4402d795701c6b308ce4",
    )).toBe(true)
  })

  it("rejects different addresses", () => {
    expect(equalAddresses(
      "0xB8cC65321d169D55b93b4402D795701c6B308ce4",
      "0x4200000000000000000000000000000000000006",
    )).toBe(false)
  })
})
