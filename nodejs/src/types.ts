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
}

export interface TxReceipt {
  blockNumber: bigint
  gasUsed: bigint
}

export interface PendingTx {
  txHash: Hex
  wait(): Promise<TxReceipt>
}

export interface PendingSwap {
  txHash: Hex
  wait(): Promise<SwapResult>
}

export interface Token {
  address: Address
  symbol: string
  decimals: number
  active: boolean
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
}

export interface SwapResult {
  txHash: Hex
  blockNumber: bigint
  /** Actual amountIn from SwapExecuted event */
  amountIn: bigint
  /** Actual amountOut from SwapExecuted event */
  amountOut: bigint
  tokenIn: Address
  tokenOut: Address
  gasUsed: bigint
}
