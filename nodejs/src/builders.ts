import { encodePacked, size } from "viem"
import { ZERO_ADDRESS } from "./address.js"
import type { Step } from "./encode-steps.js"
import type { Address, Hex } from "./types.js"

/** Route IDs registered in RouteRegistry on every chain (see DeployInfra). */
export const ROUTE_ID = {
  CURVE128:         1,
  CAKE_V3:          2,
  UNI_V3:           3,
  UNI_V4:           4,
  AERODROME:        5,
  CURVE256:         6,
  FLUID:            7,
  AAVE_LIQUIDATOR:  8,
  BALANCER_V3:      9,
} as const

// ─── Uniswap V3 ──────────────────────────────────────────────────────────────
//
// stepData layout (abi.encodePacked) — 59 bytes:
//   address tokenOut          (20)
//   uint24  fee               ( 3)
//   uint128 minOut            (16)
//   uint160 sqrtPriceLimitX96 (20)

export interface BuildUniV3StepInput {
  tokenOut: Address
  fee: number
  minOut: bigint
  sqrtPriceLimitX96?: bigint
}

export function buildUniV3Step(input: BuildUniV3StepInput): Step {
  const data = encodePacked(
    ["address", "uint24", "uint128", "uint160"],
    [input.tokenOut, input.fee, input.minOut, input.sqrtPriceLimitX96 ?? 0n],
  )
  return { id: ROUTE_ID.UNI_V3, data }
}

// ─── PancakeSwap V3 (CakeV3) ─────────────────────────────────────────────────
// Same layout as UniV3 — 59 bytes.

export function buildCakeV3Step(input: BuildUniV3StepInput): Step {
  const data = encodePacked(
    ["address", "uint24", "uint128", "uint160"],
    [input.tokenOut, input.fee, input.minOut, input.sqrtPriceLimitX96 ?? 0n],
  )
  return { id: ROUTE_ID.CAKE_V3, data }
}

// ─── Uniswap V4 ──────────────────────────────────────────────────────────────
//
// stepData layout — 83 bytes:
//   address currency0    (20)
//   address currency1    (20)
//   uint24  fee          ( 3)
//   uint24  tickSpacing  ( 3)   (encoded as uint24 in calldata, parsed as int24 on-chain)
//   address hooks        (20)
//   uint8   zeroForOne   ( 1)
//   uint128 minOut       (16)

export interface BuildUniV4StepInput {
  currency0: Address
  currency1: Address
  fee: number
  /** v4 tickSpacing is int24 on-chain. Pass positive values (-typed callers should convert). */
  tickSpacing: number
  hooks: Address
  zeroForOne: boolean
  minOut: bigint
}

export function buildUniV4Step(input: BuildUniV4StepInput): Step {
  // The on-chain reader does `shr(232, calldataload(o+43))` over the 3-byte slot,
  // i.e. unsigned 24-bit. Encode with int24 to preserve signedness (range -2^23..2^23-1).
  const data = encodePacked(
    ["address", "address", "uint24", "int24", "address", "uint8", "uint128"],
    [
      input.currency0,
      input.currency1,
      input.fee,
      input.tickSpacing,
      input.hooks,
      input.zeroForOne ? 1 : 0,
      input.minOut,
    ],
  )
  return { id: ROUTE_ID.UNI_V4, data }
}

// ─── Aerodrome (Slipstream) ──────────────────────────────────────────────────
//
// stepData layout — 59 bytes:
//   address tokenOut          (20)
//   int24   tickSpacing       ( 3)   SIGNED
//   uint160 sqrtPriceLimitX96 (20)
//   uint128 minOut            (16)

export interface BuildAerodromeStepInput {
  tokenOut: Address
  /** SIGNED int24. Aerodrome stable pools use small positive values; volatile pools can be negative on other chains. */
  tickSpacing: number
  sqrtLimit?: bigint
  minOut: bigint
}

export function buildAerodromeStep(input: BuildAerodromeStepInput): Step {
  const data = encodePacked(
    ["address", "int24", "uint160", "uint128"],
    [input.tokenOut, input.tickSpacing, input.sqrtLimit ?? 0n, input.minOut],
  )
  return { id: ROUTE_ID.AERODROME, data }
}

// ─── Balancer V3 ─────────────────────────────────────────────────────────────
//
// stepData layout — 56 bytes:
//   address pool      (20)
//   address tokenOut  (20)
//   uint128 minOut    (16)

export interface BuildBalancerV3StepInput {
  pool: Address
  tokenOut: Address
  minOut: bigint
}

export function buildBalancerV3Step(input: BuildBalancerV3StepInput): Step {
  const data = encodePacked(
    ["address", "address", "uint128"],
    [input.pool, input.tokenOut, input.minOut],
  )
  return { id: ROUTE_ID.BALANCER_V3, data }
}

// ─── Fluid DEX ───────────────────────────────────────────────────────────────
//
// stepData layout — 57 bytes:
//   address pool      (20)
//   bool    swap0to1  ( 1)
//   address tokenOut  (20)
//   uint128 minOut    (16)

export interface BuildFluidStepInput {
  pool: Address
  swap0to1: boolean
  tokenOut: Address
  minOut: bigint
}

export function buildFluidStep(input: BuildFluidStepInput): Step {
  const data = encodePacked(
    ["address", "bool", "address", "uint128"],
    [input.pool, input.swap0to1, input.tokenOut, input.minOut],
  )
  return { id: ROUTE_ID.FLUID, data }
}

// ─── Curve (uint128 indices in pool ABI) ────────────────────────────────────
//
// stepData layout — 58 bytes:
//   uint8   i         ( 1)
//   uint8   j         ( 1)
//   uint128 minDy     (16)
//   address pool      (20)
//   address tokenOut  (20)

export interface BuildCurveStepInput {
  i: number
  j: number
  minDy: bigint
  pool: Address
  tokenOut: Address
}

export function buildCurve128Step(input: BuildCurveStepInput): Step {
  const data = encodePacked(
    ["uint8", "uint8", "uint128", "address", "address"],
    [input.i, input.j, input.minDy, input.pool, input.tokenOut],
  )
  return { id: ROUTE_ID.CURVE128, data }
}

export function buildCurve256Step(input: BuildCurveStepInput): Step {
  const data = encodePacked(
    ["uint8", "uint8", "uint128", "address", "address"],
    [input.i, input.j, input.minDy, input.pool, input.tokenOut],
  )
  return { id: ROUTE_ID.CURVE256, data }
}

// ─── Aave Liquidator ─────────────────────────────────────────────────────────
//
// stepData layout — 60 bytes:
//   address pool             (20)
//   address user             (20)  — borrower being liquidated
//   address collateralAsset  (20)

export interface BuildAaveLiquidatorStepInput {
  pool: Address
  user: Address
  collateralAsset: Address
}

export function buildAaveLiquidatorStep(input: BuildAaveLiquidatorStepInput): Step {
  const data = encodePacked(
    ["address", "address", "address"],
    [input.pool, input.user, input.collateralAsset],
  )
  return { id: ROUTE_ID.AAVE_LIQUIDATOR, data }
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

/**
 * Returns the raw byte count of an encoded step's `data` payload.
 * Useful for tests that assert layout sizes match the Solidity comments.
 */
export function stepDataByteLength(data: Hex): number {
  return size(data)
}

/** Re-exported for callers that want to set `hooks` / `tokenOut` to address(0). */
export { ZERO_ADDRESS }
