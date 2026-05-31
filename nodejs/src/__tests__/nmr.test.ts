import { describe, expect, it } from "vitest"
import { decodeFunctionData, toFunctionSelector } from "viem"
import { NMR_ABI } from "../constants.js"
import {
  encodeNMRLoan,
  encodeNMRRequestOperation,
  encodeNMRSetTreasury,
  encodeNMRSweepProfit,
  encodeNMRSwap,
} from "../nmr.js"

const ASSET    = "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913" as const
const USER     = "0x4200000000000000000000000000000000000006" as const
const TREASURY = "0x1111111111111111111111111111111111111111" as const

function selectorFor(signature: string): string {
  return toFunctionSelector(signature)
}

function assertHexCall(data: string, fnName: string) {
  expect(data).toMatch(/^0x[0-9a-f]+$/i)
  // 4-byte selector = 10 hex chars including 0x
  expect(data.length).toBeGreaterThanOrEqual(10)
  const decoded = decodeFunctionData({ abi: NMR_ABI, data: data as `0x${string}` })
  expect(decoded.functionName).toBe(fnName)
}

describe("encodeNMRRequestOperation", () => {
  it("encodes requestOperation(address,uint256,bytes)", () => {
    const data = encodeNMRRequestOperation(ASSET, 1_000n, "0xdead")
    assertHexCall(data, "requestOperation")
    expect(data.startsWith(selectorFor("requestOperation(address,uint256,bytes)"))).toBe(true)
  })
})

describe("encodeNMRSwap", () => {
  it("encodes swap(address,uint256,uint256,bytes)", () => {
    const data = encodeNMRSwap(ASSET, 1_000n, 900n, "0xbeef")
    assertHexCall(data, "swap")
    expect(data.startsWith(selectorFor("swap(address,uint256,uint256,bytes)"))).toBe(true)
  })
})

describe("encodeNMRLoan", () => {
  it("encodes loan(address,address,uint256,uint256,bytes)", () => {
    const data = encodeNMRLoan(USER, ASSET, 1_000n, 900n, "0xcafe")
    assertHexCall(data, "loan")
    expect(data.startsWith(selectorFor("loan(address,address,uint256,uint256,bytes)"))).toBe(true)
  })
})

describe("encodeNMRSweepProfit", () => {
  it("encodes sweepProfit(address,uint256)", () => {
    const data = encodeNMRSweepProfit(ASSET, 42n)
    assertHexCall(data, "sweepProfit")
    expect(data.startsWith(selectorFor("sweepProfit(address,uint256)"))).toBe(true)
  })
})

describe("encodeNMRSetTreasury", () => {
  it("encodes setTreasury(address)", () => {
    const data = encodeNMRSetTreasury(TREASURY)
    assertHexCall(data, "setTreasury")
    expect(data.startsWith(selectorFor("setTreasury(address)"))).toBe(true)
  })
})
