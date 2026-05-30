import { getAddress } from "viem"
import type { Address } from "./types.js"

/** Canonical all-zeros address. */
export const ZERO_ADDRESS: Address = "0x0000000000000000000000000000000000000000"

const HEX_ADDRESS = /^0x[0-9a-fA-F]{40}$/

/** True when `s` is a syntactically valid 20-byte hex address (no checksum check). */
export function isAddress(s: string): s is Address {
  return HEX_ADDRESS.test(s)
}

/**
 * Returns the EIP-55 checksummed form of `s`. Throws if `s` is not a valid address.
 *
 *     checksumAddress("0xb8cc65321d169d55b93b4402d795701c6b308ce4")
 *     // → "0xB8cC65321d169D55b93b4402D795701c6B308ce4"
 */
export function checksumAddress(s: string): Address {
  return getAddress(s) as Address
}

/** True when `s` is the zero address (case-insensitive). */
export function isZeroAddress(s: string): boolean {
  return s.toLowerCase() === ZERO_ADDRESS
}

/** Case-insensitive address equality. Returns false for invalid inputs. */
export function equalAddresses(a: string, b: string): boolean {
  return a.toLowerCase() === b.toLowerCase()
}
