import { concat, toHex } from "viem"
import type { Hex } from "./types.js"

export interface Step {
  /** uint16 route identifier. */
  id:   number
  /** Route-specific stepData payload (hex-encoded bytes). */
  data: Hex
}

/**
 * Pack steps into the AFI tight format consumed by `Lib.runRoutes` on-chain:
 *
 *   uint8 numSteps + [uint16 routeID | uint16 dataLen | bytes data]×N
 *
 * Throws on >255 steps or data >65535 bytes per step.
 */
export function encodeSteps(steps: Step[]): Hex {
  if (steps.length > 255) {
    throw new Error(`encodeSteps: ${steps.length} steps exceeds u8 max`)
  }

  const parts: Hex[] = [toHex(steps.length, { size: 1 })]
  for (const s of steps) {
    const dataBytes = s.data.slice(2).length / 2 // hex chars to byte count
    if (dataBytes > 65535) {
      throw new Error(`encodeSteps: data ${dataBytes} exceeds u16 max`)
    }
    parts.push(toHex(s.id, { size: 2 }))
    parts.push(toHex(dataBytes, { size: 2 }))
    parts.push(s.data)
  }
  return concat(parts)
}
