import { decodeErrorResult } from "viem"
import { AFI_ABI } from "./constants.js"
import type { Hex } from "./types.js"

interface AbiError {
  type: "error"
  name: string
  inputs: ReadonlyArray<{ name?: string; type: string }>
}

/**
 * Structured revert reason decoded from raw revert data.
 *
 *     DecodedRevert { name: "InsufficientFunds", signature: "InsufficientFunds(uint256)", args: [100n] }
 */
export interface DecodedRevert {
  /** Error name (e.g. "InsufficientFunds"). */
  name: string
  /** Full signature including parameter types (e.g. "InsufficientFunds(uint256)"). */
  signature: string
  /** Decoded arguments in declaration order. */
  args: readonly unknown[]
}

// Selectors for the two solidity built-in revert variants.
const ERROR_STRING_SELECTOR = "0x08c379a0"  // Error(string)
const PANIC_SELECTOR        = "0x4e487b71"  // Panic(uint256)

// User-registered custom error ABIs are decoded in addition to AFI_ABI.
const userErrors: AbiError[] = []

/**
 * Registers additional custom errors so `decodeRevertReason` can decode them.
 * Pass an ABI array (or just the error entries) from your own contracts.
 */
export function registerCustomErrors(abi: readonly unknown[]): void {
  for (const entry of abi) {
    const e = entry as { type?: string }
    if (e.type === "error") userErrors.push(entry as AbiError)
  }
}

/** Returns the currently registered user-provided error definitions. */
export function getRegisteredErrors(): readonly AbiError[] {
  return userErrors
}

/**
 * Decodes raw revert data (hex) into a structured error.
 *
 * Tries, in order:
 *  1. Standard `Error(string)`  (selector 0x08c379a0)
 *  2. Standard `Panic(uint256)` (selector 0x4e487b71)
 *  3. AFI custom errors from AFI_ABI
 *  4. User-registered errors (via `registerCustomErrors`)
 *
 * Returns `null` when the data is empty or doesn't match any known error.
 */
export function decodeRevertReason(data?: Hex | string | null): DecodedRevert | null {
  if (!data || typeof data !== "string" || data.length < 10) return null
  const hex = (data.startsWith("0x") ? data : `0x${data}`) as Hex

  // Compose a registry: AFI errors + standards + user-registered.
  const allErrors: AbiError[] = [
    ...(AFI_ABI.filter((e) => (e as { type?: string }).type === "error") as unknown as AbiError[]),
    ...userErrors,
    { type: "error", name: "Error", inputs: [{ name: "message", type: "string" }] },
    { type: "error", name: "Panic", inputs: [{ name: "code", type: "uint256" }] },
  ]

  try {
    const r = decodeErrorResult({ abi: allErrors as never, data: hex })
    const matched = allErrors.find((e) => e.name === r.errorName)
    return formatDecoded(r.errorName, r.args ?? [], matched)
  } catch {
    return null
  }
}

// Selectors for built-in Solidity revert variants — kept exported for tests/diagnostics.
export const ERROR_STRING_SELECTOR_HEX = ERROR_STRING_SELECTOR
export const PANIC_SELECTOR_HEX        = PANIC_SELECTOR

function formatDecoded(name: string, args: readonly unknown[], def?: AbiError): DecodedRevert {
  const typeSig = def?.inputs?.map((i) => i.type).join(",") ?? ""
  return {
    name,
    signature: `${name}(${typeSig})`,
    args,
  }
}

/** Human-friendly one-line summary of a decoded revert ("InsufficientFunds: available=100"). */
export function describeDecodedRevert(d: DecodedRevert | null | undefined): string {
  if (!d) return "unknown revert"
  if (d.args.length === 0) return d.name
  const args = d.args.map(v => (typeof v === "bigint" ? v.toString() : String(v))).join(", ")
  return `${d.name}(${args})`
}
