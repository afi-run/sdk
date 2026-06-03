import { describe, expect, it } from "vitest"
import { encodeAbiParameters, toFunctionSelector } from "viem"
import {
  decodeRevertReason,
  describeDecodedRevert,
  ERROR_STRING_SELECTOR_HEX,
  getRegisteredErrors,
  PANIC_SELECTOR_HEX,
  registerCustomErrors,
} from "../decode.js"

function encodeRevert(signature: string, paramTypes: string[], values: readonly unknown[]): `0x${string}` {
  const selector = toFunctionSelector(signature)
  if (paramTypes.length === 0) return selector
  const payload = encodeAbiParameters(
    paramTypes.map((type) => ({ type })),
    values as never[],
  )
  return (selector + payload.slice(2)) as `0x${string}`
}

describe("decodeRevertReason — built-in revert", () => {
  it("decodes Error(string) reverts", () => {
    const data = encodeRevert("Error(string)", ["string"], ["boom"])
    const r = decodeRevertReason(data)
    expect(r).not.toBeNull()
    expect(r!.name).toBe("Error")
    expect(r!.args[0]).toBe("boom")
    expect(ERROR_STRING_SELECTOR_HEX).toBe("0x08c379a0")
  })

  it("decodes Panic(uint256)", () => {
    const data = encodeRevert("Panic(uint256)", ["uint256"], [0x11n])
    const r = decodeRevertReason(data)
    expect(r!.name).toBe("Panic")
    expect(r!.args[0]).toBe(0x11n)
    expect(PANIC_SELECTOR_HEX).toBe("0x4e487b71")
  })
})

describe("decodeRevertReason — AFI custom errors", () => {
  it("decodes InsufficientFunds(uint256)", () => {
    const data = encodeRevert("InsufficientFunds(uint256)", ["uint256"], [1234n])
    const r = decodeRevertReason(data)
    expect(r).not.toBeNull()
    expect(r!.name).toBe("InsufficientFunds")
    expect(r!.signature).toBe("InsufficientFunds(uint256)")
    expect(r!.args[0]).toBe(1234n)
  })

  it("decodes ZeroAddress() with no args", () => {
    const data = encodeRevert("ZeroAddress()", [], [])
    const r = decodeRevertReason(data)
    expect(r!.name).toBe("ZeroAddress")
    expect(r!.args).toHaveLength(0)
  })

  it("decodes DifferentAssets(address,address)", () => {
    const expected = "0x4200000000000000000000000000000000000006"
    const actual = "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"
    const data = encodeRevert("DifferentAssets(address,address)", ["address", "address"], [expected, actual])
    const r = decodeRevertReason(data)
    expect(r!.name).toBe("DifferentAssets")
    expect((r!.args[0] as string).toLowerCase()).toBe(expected)
    expect((r!.args[1] as string).toLowerCase()).toBe(actual)
  })

  it("decodes ReentrancyGuardReentrantCall (OpenZeppelin)", () => {
    const data = encodeRevert("ReentrancyGuardReentrantCall()", [], [])
    expect(decodeRevertReason(data)!.name).toBe("ReentrancyGuardReentrantCall")
  })

  it("decodes OwnableUnauthorizedAccount(address)", () => {
    const data = encodeRevert("OwnableUnauthorizedAccount(address)", ["address"], ["0xdeadbeef00000000000000000000000000000000"])
    expect(decodeRevertReason(data)!.name).toBe("OwnableUnauthorizedAccount")
  })
})

describe("decodeRevertReason — fallbacks", () => {
  it("returns null for empty or too-short data", () => {
    expect(decodeRevertReason(undefined)).toBeNull()
    expect(decodeRevertReason(null)).toBeNull()
    expect(decodeRevertReason("0x")).toBeNull()
    expect(decodeRevertReason("0x1234")).toBeNull()
  })

  it("returns null for unknown selectors", () => {
    const data = encodeRevert("Mystery(uint256)", ["uint256"], [1n])
    expect(decodeRevertReason(data)).toBeNull()
  })

  it("accepts hex data without the 0x prefix", () => {
    const data = encodeRevert("Error(string)", ["string"], ["boom"])
    const r = decodeRevertReason(data.slice(2)) // strip "0x"
    expect(r).not.toBeNull()
    expect(r!.name).toBe("Error")
    expect(r!.args[0]).toBe("boom")
  })

  it("returns null for non-string data", () => {
    // @ts-expect-error exercising the runtime guard
    expect(decodeRevertReason(12345)).toBeNull()
  })
})

describe("registerCustomErrors + getRegisteredErrors", () => {
  it("ignores non-error ABI entries when registering", () => {
    const before = getRegisteredErrors().length
    registerCustomErrors([
      { type: "function", name: "ignored", inputs: [] },
      { type: "event", name: "alsoIgnored", inputs: [] },
    ])
    expect(getRegisteredErrors().length).toBe(before)
  })

  it("decodes user-registered errors after registration", () => {
    const data = encodeRevert("MyContractError(uint256,string)", ["uint256", "string"], [42n, "details"])

    expect(decodeRevertReason(data)).toBeNull()

    registerCustomErrors([
      {
        type: "error",
        name: "MyContractError",
        inputs: [
          { name: "code", type: "uint256" },
          { name: "msg", type: "string" },
        ],
      },
    ])

    const r = decodeRevertReason(data)
    expect(r).not.toBeNull()
    expect(r!.name).toBe("MyContractError")
    expect(r!.args[0]).toBe(42n)
    expect(r!.args[1]).toBe("details")
  })
})

describe("describeDecodedRevert", () => {
  it("formats with args", () => {
    expect(describeDecodedRevert({ name: "InsufficientFunds", signature: "InsufficientFunds(uint256)", args: [100n] }))
      .toBe("InsufficientFunds(100)")
  })

  it("formats no-arg errors as just the name", () => {
    expect(describeDecodedRevert({ name: "ZeroAddress", signature: "ZeroAddress()", args: [] }))
      .toBe("ZeroAddress")
  })

  it("stringifies mixed bigint and non-bigint args", () => {
    expect(describeDecodedRevert({
      name: "Mixed",
      signature: "Mixed(uint256,address)",
      args: [100n, "0xabc"],
    })).toBe("Mixed(100, 0xabc)")
  })

  it("handles undefined", () => {
    expect(describeDecodedRevert(undefined)).toBe("unknown revert")
  })

  it("handles null", () => {
    expect(describeDecodedRevert(null)).toBe("unknown revert")
  })
})
