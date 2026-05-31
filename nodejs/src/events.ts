import { decodeEventLog, type Log } from "viem"
import { AFI_ABI, NMR_ABI } from "./constants.js"
import type { Address, Hex } from "./types.js"

type AnyAbi = readonly unknown[]

/**
 * Try to decode a log against an ABI looking for a specific event. Returns
 * `undefined` if the log doesn't match (different topic0, malformed, etc).
 */
function tryDecode<T>(log: Log, abi: AnyAbi, eventName: string): T | undefined {
  if (!log.topics || log.topics.length === 0) return undefined
  try {
    const decoded = decodeEventLog({
      abi: abi as never,
      data: log.data,
      topics: log.topics as [signature: `0x${string}`, ...args: `0x${string}`[]],
      strict: false,
    })
    if (decoded.eventName !== eventName) return undefined
    return decoded.args as unknown as T
  } catch {
    return undefined
  }
}

function parseAll<T>(logs: Log[], abi: AnyAbi, eventName: string): T[] {
  const out: T[] = []
  for (const log of logs) {
    const parsed = tryDecode<T>(log, abi, eventName)
    if (parsed !== undefined) out.push(parsed)
  }
  return out
}

// ─── Afi events ────────────────────────────────────────────────────────────────

export interface SwapExecutedEvent {
  from: Address
  assetIn: Address
  amountIn: bigint
  assetOut: Address
  amountOut: bigint
}

export function parseSwapExecuted(logs: Log[]): SwapExecutedEvent[] {
  return parseAll<SwapExecutedEvent>(logs, AFI_ABI, "SwapExecuted")
}

export interface FeeCollectedEvent {
  token: Address
  amount: bigint
}

export function parseFeeCollected(logs: Log[]): FeeCollectedEvent[] {
  return parseAll<FeeCollectedEvent>(logs, AFI_ABI, "FeeCollected")
}

export interface TreasuryUpdatedEvent {
  treasury: Address
}

export function parseTreasuryUpdated(logs: Log[]): TreasuryUpdatedEvent[] {
  return parseAll<TreasuryUpdatedEvent>(logs, AFI_ABI, "TreasuryUpdated")
}

export interface FeeBpsUpdatedEvent {
  feeBps: number
}

export function parseFeeBpsUpdated(logs: Log[]): FeeBpsUpdatedEvent[] {
  return parseAll<FeeBpsUpdatedEvent>(logs, AFI_ABI, "FeeBpsUpdated")
}

export interface UserFeeBpsSetEvent {
  user: Address
  feeBps: number
}

export function parseUserFeeBpsSet(logs: Log[]): UserFeeBpsSetEvent[] {
  return parseAll<UserFeeBpsSetEvent>(logs, AFI_ABI, "UserFeeBpsSet")
}

export interface UserFeeBpsClearedEvent {
  user: Address
}

export function parseUserFeeBpsCleared(logs: Log[]): UserFeeBpsClearedEvent[] {
  return parseAll<UserFeeBpsClearedEvent>(logs, AFI_ABI, "UserFeeBpsCleared")
}

// ─── NMR events ────────────────────────────────────────────────────────────────

export interface FlashLoanRequestedEvent {
  asset: Address
  amount: bigint
}

export function parseFlashLoanRequested(logs: Log[]): FlashLoanRequestedEvent[] {
  return parseAll<FlashLoanRequestedEvent>(logs, NMR_ABI, "FlashLoanRequested")
}

export interface FlashLoanExecutedEvent {
  asset: Address
  amount: bigint
  premium: bigint
  profit: bigint
}

export function parseFlashLoanExecuted(logs: Log[]): FlashLoanExecutedEvent[] {
  return parseAll<FlashLoanExecutedEvent>(logs, NMR_ABI, "FlashLoanExecuted")
}

export interface FlashLoanFailedEvent {
  asset: Address
  amount: bigint
  reason: string
}

export function parseFlashLoanFailed(logs: Log[]): FlashLoanFailedEvent[] {
  return parseAll<FlashLoanFailedEvent>(logs, NMR_ABI, "FlashLoanFailed")
}

export interface FlashLoanFailedWithDataEvent {
  asset: Address
  amount: bigint
  data: Hex
}

/**
 * Parses NMR FlashLoanFailedWithData events (low-level revert data variant).
 * Pair with parseFlashLoanFailed: the contract emits one or the other depending
 * on whether the revert carried a decodable string reason.
 */
export function parseFlashLoanFailedWithData(logs: Log[]): FlashLoanFailedWithDataEvent[] {
  return parseAll<FlashLoanFailedWithDataEvent>(logs, NMR_ABI, "FlashLoanFailedWithData")
}

export interface NmrSwapExecutedEvent {
  assetIn: Address
  amountIn: bigint
  assetOut: Address
  amountOut: bigint
}

/**
 * Parses NMR-emitted SwapExecuted events (the cycle-swap variant, distinct from
 * Afi's — no `from` field, different topic0). Use parseSwapExecuted for Afi logs.
 */
export function parseNmrSwapExecuted(logs: Log[]): NmrSwapExecutedEvent[] {
  return parseAll<NmrSwapExecutedEvent>(logs, NMR_ABI, "SwapExecuted")
}

export interface ProfitSweptEvent {
  asset: Address
  amount: bigint
  to: Address
}

export function parseProfitSwept(logs: Log[]): ProfitSweptEvent[] {
  return parseAll<ProfitSweptEvent>(logs, NMR_ABI, "ProfitSwept")
}

export interface ProfitShareUpdatedEvent {
  profitShare: number
}

export function parseProfitShareUpdated(logs: Log[]): ProfitShareUpdatedEvent[] {
  return parseAll<ProfitShareUpdatedEvent>(logs, NMR_ABI, "ProfitShareUpdated")
}
