import { describe, expect, it } from "vitest"
import { decodeFunctionData, toFunctionSelector } from "viem"
import { AFI_ABI } from "../constants.js"
import {
  encodeAcceptOwnership,
  encodeAfiPause,
  encodeAfiUnpause,
  encodeAfiSetTreasury,
  encodeAfiSetFeeBps,
  encodeAfiSetUserFeeBps,
  encodeAfiSetUserFeeBpsBatch,
  encodeAfiClearUserFeeBps,
  encodeAfiResetAnyUserOverride,
  encodeAfiAddRule,
  encodeAfiClearRules,
  encodeAfiSetOperator,
  encodeAfiRescueTokens,
} from "../afi-admin.js"

const TREASURY = "0x1111111111111111111111111111111111111111" as const
const USER     = "0x2222222222222222222222222222222222222222" as const
const USER2    = "0x3333333333333333333333333333333333333333" as const
const RULE     = "0x4444444444444444444444444444444444444444" as const
const OPERATOR = "0x5555555555555555555555555555555555555555" as const
const TOKEN    = "0x6666666666666666666666666666666666666666" as const

function assertCall(data: string, fnName: string, signature: string) {
  expect(data).toMatch(/^0x[0-9a-f]+$/i)
  expect(data.length).toBeGreaterThanOrEqual(10)
  expect(data.startsWith(toFunctionSelector(signature))).toBe(true)
  const decoded = decodeFunctionData({ abi: AFI_ABI, data: data as `0x${string}` })
  expect(decoded.functionName).toBe(fnName)
  return decoded
}

describe("encodeAfiPause", () => {
  it("encodes pause()", () => {
    const data = encodeAfiPause()
    assertCall(data, "pause", "pause()")
  })
})

describe("encodeAfiUnpause", () => {
  it("encodes unpause()", () => {
    const data = encodeAfiUnpause()
    assertCall(data, "unpause", "unpause()")
  })
})

describe("encodeAfiSetTreasury", () => {
  it("encodes setTreasury(address)", () => {
    const data = encodeAfiSetTreasury(TREASURY)
    const decoded = assertCall(data, "setTreasury", "setTreasury(address)")
    expect((decoded.args as readonly unknown[])[0]).toBe(TREASURY)
  })
})

describe("encodeAfiSetFeeBps", () => {
  it("encodes setFeeBps(uint16)", () => {
    const data = encodeAfiSetFeeBps(42)
    const decoded = assertCall(data, "setFeeBps", "setFeeBps(uint16)")
    expect((decoded.args as readonly unknown[])[0]).toBe(42)
  })
})

describe("encodeAfiSetUserFeeBps", () => {
  it("encodes setUserFeeBps(address,uint16)", () => {
    const data = encodeAfiSetUserFeeBps(USER, 25)
    const decoded = assertCall(data, "setUserFeeBps", "setUserFeeBps(address,uint16)")
    const args = decoded.args as readonly unknown[]
    expect(args[0]).toBe(USER)
    expect(args[1]).toBe(25)
  })
})

describe("encodeAfiSetUserFeeBpsBatch", () => {
  it("encodes setUserFeeBpsBatch(address[],uint16[])", () => {
    const data = encodeAfiSetUserFeeBpsBatch([USER, USER2], [10, 20])
    const decoded = assertCall(
      data,
      "setUserFeeBpsBatch",
      "setUserFeeBpsBatch(address[],uint16[])",
    )
    const args = decoded.args as readonly unknown[]
    expect(args[0]).toEqual([USER, USER2])
    expect(args[1]).toEqual([10, 20])
  })
})

describe("encodeAfiClearUserFeeBps", () => {
  it("encodes clearUserFeeBps(address)", () => {
    const data = encodeAfiClearUserFeeBps(USER)
    const decoded = assertCall(data, "clearUserFeeBps", "clearUserFeeBps(address)")
    expect((decoded.args as readonly unknown[])[0]).toBe(USER)
  })
})

describe("encodeAfiResetAnyUserOverride", () => {
  it("encodes resetAnyUserOverride()", () => {
    const data = encodeAfiResetAnyUserOverride()
    assertCall(data, "resetAnyUserOverride", "resetAnyUserOverride()")
  })
})

describe("encodeAfiAddRule", () => {
  it("encodes addRule(address)", () => {
    const data = encodeAfiAddRule(RULE)
    const decoded = assertCall(data, "addRule", "addRule(address)")
    expect((decoded.args as readonly unknown[])[0]).toBe(RULE)
  })
})

describe("encodeAfiClearRules", () => {
  it("encodes clearRules()", () => {
    const data = encodeAfiClearRules()
    assertCall(data, "clearRules", "clearRules()")
  })
})

describe("encodeAfiSetOperator", () => {
  it("encodes setOperator(address,bool)", () => {
    const data = encodeAfiSetOperator(OPERATOR, true)
    const decoded = assertCall(data, "setOperator", "setOperator(address,bool)")
    const args = decoded.args as readonly unknown[]
    expect(args[0]).toBe(OPERATOR)
    expect(args[1]).toBe(true)
  })
})

describe("encodeAfiRescueTokens", () => {
  it("encodes rescueTokens(address,uint256,address)", () => {
    const data = encodeAfiRescueTokens(TOKEN, 1_000n, TREASURY)
    const decoded = assertCall(data, "rescueTokens", "rescueTokens(address,uint256,address)")
    const args = decoded.args as readonly unknown[]
    expect(args[0]).toBe(TOKEN)
    expect(args[1]).toBe(1_000n)
    expect(args[2]).toBe(TREASURY)
  })
})

describe("encodeAcceptOwnership", () => {
  it("produces the Ownable2Step acceptOwnership() selector (0x79ba5097)", () => {
    const data = encodeAcceptOwnership()
    expect(data).toBe("0x79ba5097")
    expect(data.startsWith(toFunctionSelector("acceptOwnership()"))).toBe(true)
  })
})

const ZERO = "0x0000000000000000000000000000000000000000" as const

describe("input validations", () => {
  it("rejects feeBps > MAX_FEE_BPS (50)", () => {
    expect(() => encodeAfiSetFeeBps(51)).toThrow(/in \[0, 50\]/)
  })

  it("rejects negative feeBps", () => {
    expect(() => encodeAfiSetFeeBps(-1)).toThrow(/in \[0, 50\]/)
  })

  it("accepts feeBps at MAX_FEE_BPS boundary", () => {
    expect(() => encodeAfiSetFeeBps(50)).not.toThrow()
    expect(() => encodeAfiSetFeeBps(0)).not.toThrow()
  })

  it("rejects setUserFeeBps over range", () => {
    expect(() => encodeAfiSetUserFeeBps(USER, 51)).toThrow(/in \[0, 50\]/)
  })

  it("rejects batch with mismatched lengths", () => {
    expect(() => encodeAfiSetUserFeeBpsBatch([USER, USER2], [1])).toThrow(/must equal/)
  })

  it("rejects batch with any over-range entry", () => {
    expect(() => encodeAfiSetUserFeeBpsBatch([USER, USER2], [10, 51])).toThrow(/in \[0, 50\]/)
  })

  it("rejects setTreasury(zero address)", () => {
    expect(() => encodeAfiSetTreasury(ZERO)).toThrow(/zero address/)
  })

  it("rejects addRule(zero address)", () => {
    expect(() => encodeAfiAddRule(ZERO)).toThrow(/zero address/)
  })
})
