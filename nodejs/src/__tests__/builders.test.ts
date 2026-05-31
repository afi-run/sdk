import { describe, expect, it } from "vitest"
import { size } from "viem"
import {
  ROUTE_ID,
  buildAaveLiquidatorStep,
  buildAerodromeStep,
  buildBalancerV3Step,
  buildCakeV3Step,
  buildCurve128Step,
  buildCurve256Step,
  buildFluidStep,
  buildUniV3Step,
  buildUniV4Step,
  stepDataByteLength,
} from "../builders.js"

const TOKEN_OUT = "0x4200000000000000000000000000000000000006" as const
const POOL      = "0x1111111111111111111111111111111111111111" as const
const USER      = "0x2222222222222222222222222222222222222222" as const
const COLLATERAL = "0x3333333333333333333333333333333333333333" as const
const HOOKS     = "0x0000000000000000000000000000000000000000" as const

describe("builders — UniV3", () => {
  it("encodes a 59-byte stepData with id=3", () => {
    const step = buildUniV3Step({ tokenOut: TOKEN_OUT, fee: 500, minOut: 1n })
    expect(step.id).toBe(ROUTE_ID.UNI_V3)
    expect(step.id).toBe(3)
    expect(size(step.data)).toBe(59)
    // 20 (addr) + 3 (fee=0x0001f4) + 16 (minOut=0x..01) + 20 (sqrt=0)
    expect(step.data.slice(2, 2 + 40)).toBe(TOKEN_OUT.slice(2).toLowerCase())
    expect(step.data.slice(2 + 40, 2 + 40 + 6)).toBe("0001f4")
  })

  it("uses 0 as default sqrtPriceLimit", () => {
    const step = buildUniV3Step({ tokenOut: TOKEN_OUT, fee: 0, minOut: 0n })
    // Last 20 bytes (40 hex chars) should all be zero.
    expect(step.data.slice(-40)).toBe("0".repeat(40))
  })
})

describe("builders — CakeV3", () => {
  it("encodes a 59-byte stepData with id=2", () => {
    const step = buildCakeV3Step({ tokenOut: TOKEN_OUT, fee: 100, minOut: 1n })
    expect(step.id).toBe(ROUTE_ID.CAKE_V3)
    expect(step.id).toBe(2)
    expect(size(step.data)).toBe(59)
  })
})

describe("builders — UniV4", () => {
  it("encodes an 83-byte stepData with id=4", () => {
    const step = buildUniV4Step({
      currency0:   "0x0000000000000000000000000000000000000000",
      currency1:   TOKEN_OUT,
      fee:         3000,
      tickSpacing: 60,
      hooks:       HOOKS,
      zeroForOne:  true,
      minOut:      1_000n,
    })
    expect(step.id).toBe(ROUTE_ID.UNI_V4)
    expect(step.id).toBe(4)
    expect(size(step.data)).toBe(83)
  })

  it("preserves zeroForOne=false as 0x00 byte", () => {
    const step = buildUniV4Step({
      currency0:   TOKEN_OUT,
      currency1:   "0x0000000000000000000000000000000000000000",
      fee:         3000,
      tickSpacing: 60,
      hooks:       HOOKS,
      zeroForOne:  false,
      minOut:      1n,
    })
    // zeroForOne byte sits at offset 66 (= 20+20+3+3+20)
    expect(step.data.slice(2 + 66 * 2, 2 + 66 * 2 + 2)).toBe("00")
  })
})

describe("builders — Aerodrome", () => {
  it("encodes a 59-byte stepData with id=5", () => {
    const step = buildAerodromeStep({ tokenOut: TOKEN_OUT, tickSpacing: 50, minOut: 1n })
    expect(step.id).toBe(ROUTE_ID.AERODROME)
    expect(step.id).toBe(5)
    expect(size(step.data)).toBe(59)
  })

  it("encodes negative tickSpacing as signed int24 (two's complement)", () => {
    // -1 in int24 = 0xffffff
    const step = buildAerodromeStep({ tokenOut: TOKEN_OUT, tickSpacing: -1, minOut: 0n })
    // tickSpacing byte slot: offset 20..23
    const slot = step.data.slice(2 + 20 * 2, 2 + 23 * 2)
    expect(slot).toBe("ffffff")
  })
})

describe("builders — BalancerV3", () => {
  it("encodes a 56-byte stepData with id=9", () => {
    const step = buildBalancerV3Step({ pool: POOL, tokenOut: TOKEN_OUT, minOut: 42n })
    expect(step.id).toBe(ROUTE_ID.BALANCER_V3)
    expect(step.id).toBe(9)
    expect(size(step.data)).toBe(56)
  })
})

describe("builders — Fluid", () => {
  it("encodes a 57-byte stepData with id=7", () => {
    const step = buildFluidStep({ pool: POOL, swap0to1: true, tokenOut: TOKEN_OUT, minOut: 1n })
    expect(step.id).toBe(ROUTE_ID.FLUID)
    expect(step.id).toBe(7)
    expect(size(step.data)).toBe(57)
    // swap0to1=true at offset 20
    expect(step.data.slice(2 + 20 * 2, 2 + 21 * 2)).toBe("01")
  })

  it("encodes swap0to1=false as 0x00 byte", () => {
    const step = buildFluidStep({ pool: POOL, swap0to1: false, tokenOut: TOKEN_OUT, minOut: 1n })
    expect(step.data.slice(2 + 20 * 2, 2 + 21 * 2)).toBe("00")
  })
})

describe("builders — Curve128", () => {
  it("encodes a 58-byte stepData with id=1", () => {
    const step = buildCurve128Step({ i: 0, j: 1, minDy: 0n, pool: POOL, tokenOut: TOKEN_OUT })
    expect(step.id).toBe(ROUTE_ID.CURVE128)
    expect(step.id).toBe(1)
    expect(size(step.data)).toBe(58)
    expect(step.data.slice(2, 2 + 4)).toBe("0001")
  })
})

describe("builders — Curve256", () => {
  it("encodes a 58-byte stepData with id=6", () => {
    const step = buildCurve256Step({ i: 0, j: 2, minDy: 0n, pool: POOL, tokenOut: TOKEN_OUT })
    expect(step.id).toBe(ROUTE_ID.CURVE256)
    expect(step.id).toBe(6)
    expect(size(step.data)).toBe(58)
  })
})

describe("builders — AaveLiquidator", () => {
  it("encodes a 60-byte stepData with id=8", () => {
    const step = buildAaveLiquidatorStep({ pool: POOL, user: USER, collateralAsset: COLLATERAL })
    expect(step.id).toBe(ROUTE_ID.AAVE_LIQUIDATOR)
    expect(step.id).toBe(8)
    expect(size(step.data)).toBe(60)
  })
})

describe("stepDataByteLength", () => {
  it("returns the byte length of the data payload", () => {
    const step = buildUniV3Step({ tokenOut: TOKEN_OUT, fee: 500, minOut: 1n })
    expect(stepDataByteLength(step.data)).toBe(59)
  })
})
