export type Address = `0x${string}`
export type Hex = `0x${string}`

export const NETWORK = {
  BASE:     "base",
  BSC:      "bsc",
  ARBITRUM: "arbitrum",
  ETHEREUM: "ethereum",
  UNICHAIN: "unichain",
} as const
export type Network = typeof NETWORK[keyof typeof NETWORK]

export const DEX = {
  UNI_V3:    "uni-v3",
  UNI_V4:    "uni-v4",
  CAKE_V3:   "cake-v3",
  AERODROME: "aerodrome",
  BALANCER:  "balancer",
  CURVE128:  "curve128",
  CURVE256:  "curve256",
  FLUID:     "fluid",
} as const
export type Dex = typeof DEX[keyof typeof DEX]

export interface RpcUrlInfo {
  url: string
  account?: number
  ipc?: boolean
}

export interface AfiConfig {
  rpcUrl: string
  privateKey?: Hex
  /**
   * Percentage added on top of estimated gas for write txs (approve, swap).
   * When omitted, the SDK uses DEFAULT_GAS_BUFFER_PERCENT (15%).
   * Pass 0 to disable the buffer entirely.
   */
  gasBufferPercent?: number
  /** Diagnostic callback fired after each significant SDK operation. */
  logger?: Logger
}

export interface LogEvent {
  /** "rpc" for chain calls, "api" for AFI HTTP API calls. */
  kind: "rpc" | "api"
  /** Operation name (e.g. "getQuote", "executeSwap", "tokenInfo"). */
  method: string
  /** Wall-clock duration in milliseconds. */
  durationMs: number
  /** True when the operation completed without error. */
  ok: boolean
  /** Error thrown by the operation (only present when `ok` is false). */
  error?: unknown
}

export type Logger = (event: LogEvent) => void

export interface WaitForTxOptions {
  /** Minimum confirmations required (default: 1). */
  confirmations?: number
  /** Timeout in milliseconds (default: no timeout). */
  timeoutMs?: number
}

export interface ExecuteOptions {
  /** Minimum confirmations to wait for after submission (default: 1). */
  confirmations?: number
  /** Timeout in milliseconds for the confirmation wait (default: no timeout). */
  timeoutMs?: number
  /**
   * Explicit nonce for the swap tx. When omitted, the SDK uses the managed
   * nonce (if `useManagedNonce()` was called) or queries the RPC.
   */
  nonce?: number
}

export interface SwapCostEstimate {
  /** Estimated gas units (before the SDK buffer). */
  gas: bigint
  /** gas * (1 + gasBufferPercent/100). */
  gasWithBuffer: bigint
  /** maxFeePerGas the SDK would use (baseFee*2 + tip). */
  gasPriceWei: bigint
  /** gasWithBuffer * gasPriceWei. */
  totalWei: bigint
  /** totalWei formatted as ETH (18 decimals). */
  totalEth: string
}

export interface HealthEndpoint {
  ok: boolean
  durationMs: number
  /** Carries the chain ID for RPC, "ok" for API; absent on error. */
  detail?: string
  error?: unknown
}

export interface HealthCheck {
  rpc: HealthEndpoint
  api: HealthEndpoint
}

/** Live exchange rate for a token pair. */
export interface TokenPrice {
  /** How many units of tokenOut you get for 1 unit of tokenIn. */
  price: string
  /** How many units of tokenIn you get for 1 unit of tokenOut. */
  inverse: string
}

/** Status of a transaction obtained without blocking. */
export type TxStatus = "pending" | "success" | "failed" | "unknown"

/** A single blocking issue identified by `preflight()`. */
export interface PreflightProblem {
  code: "INSUFFICIENT_BALANCE" | "SIMULATION_FAILED"
  message: string
}

/** Result of `client.preflight(quote)`. */
export interface PreflightReport {
  /** True when no blocking problems were found. `executeSwap` will auto-handle approval. */
  canExecute: boolean
  /** True when an approve tx is needed before the swap can execute. */
  needsApproval: boolean
  /** Blocking issues — empty when canExecute is true. */
  problems: PreflightProblem[]
  /** Current tokenIn balance of the wallet. */
  balance: bigint
  /** Current allowance granted to the AFI router by the wallet. */
  allowance: bigint
}

export interface TxReceipt {
  blockNumber: bigint
  gasUsed:     bigint
  /** Effective gas price the chain charged (wei/gas). 0n when the receipt didn't expose it. */
  effectiveGasPrice: bigint
  /** gasUsed * effectiveGasPrice. */
  feeWei: bigint
  /** feeWei formatted as ETH (18 decimals). */
  feeEth: string
}

export interface PendingTx {
  txHash: Hex
  /** Block until the tx confirms. Pass opts to require more than 1 confirmation. */
  wait(opts?: WaitForTxOptions): Promise<TxReceipt>
}

export interface PendingSwap {
  txHash: Hex
  /** Block until the swap confirms. Pass opts to require more than 1 confirmation. */
  wait(opts?: WaitForTxOptions): Promise<SwapResult>
}

export interface Token {
  address: Address
  symbol: string
  decimals: number
  active: boolean
}

export interface TokenInfo {
  address: Address
  symbol: string
  name: string
  decimals: number
  /** Owner the balance/allowance refer to. Undefined when not queried. */
  owner?: Address
  /** Balance of `owner`. Present only when owner is provided. */
  balance?: bigint
  /** Allowance granted by `owner` to the AFI contract. Present only when owner is provided. */
  allowance?: bigint
}

export interface Hop {
  tokenIn: Address
  tokenOut: Address
  amountIn: string
  amountOut: string
  minOut: string
  amountInWei: bigint
  amountOutWei: bigint
  minOutWei: bigint
  tokenInPrice: string
  tokenOutPrice: string
  slippage: number
  /** Pool protocol, e.g. "uniV3", "aerodrome" */
  type: string
  /** Routing engine kind */
  kind: string
  routeId: number
  weight: number
}

export interface Quote {
  tokenIn: Address
  tokenOut: Address
  amountIn: string
  amountOut: string
  minOut: string
  /** Exact amount to approve and pass to swap() */
  amountInWei: bigint
  /** Estimated output (informational) */
  amountOutWei: bigint
  /** Minimum output after slippage — never bypassed */
  minOutWei: bigint
  /** Encoded route steps — passed as params to Afi.swap() */
  steps: Hex
  /** Token path for the route */
  path: Address[]
  hops: Hop[]
  slippage: number
  /** Current protocol fee in basis points */
  feeBps: number
  tokenInPrice: string
  tokenOutPrice: string
  /** Present only when priceBase is set in the builder */
  tokenInBasePrice?: string
  /** Present only when priceBase is set in the builder */
  tokenOutBasePrice?: string
  /** Unix-millisecond timestamp when the quote was fetched. */
  createdAt: number
  /** Network the quote was fetched against — needed to refresh. */
  network: Network
  /** Max hops used in the routing — needed to refresh. */
  maxHops: number
  /** Base price asset used (if any) — needed to refresh. */
  priceBase?: string
  /** DEX restriction used (if any) — needed to refresh. */
  dexs?: Dex[]
}

/** Returns true when the quote is older than `maxAgeSec` seconds. */
export function isQuoteStale(quote: Quote, maxAgeSec: number): boolean {
  if (!quote.createdAt) return false
  return (Date.now() - quote.createdAt) / 1000 > maxAgeSec
}

export interface SwapResult {
  txHash:      Hex
  blockNumber: bigint
  /** Actual amountIn from SwapExecuted event */
  amountIn:  bigint
  /** Actual amountOut from SwapExecuted event */
  amountOut: bigint
  tokenIn:   Address
  tokenOut:  Address
  gasUsed:   bigint
  /** Effective gas price (wei/gas) reported by the receipt. */
  effectiveGasPrice: bigint
  /** gasUsed * effectiveGasPrice. */
  feeWei: bigint
  /** feeWei formatted as ETH (18 decimals). */
  feeEth: string
}
