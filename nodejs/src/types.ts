export type Address = `0x${string}`
export type Hex = `0x${string}`

export interface AfiConfig {
  rpcUrl: string
  privateKey: Hex
}

export interface SwapParams {
  tokenIn: Address
  tokenOut: Address
  /** Raw amount in wei */
  amountIn: bigint
  /** Slippage percentage, e.g. 0.5 for 0.5% */
  slippage: number
}

export interface Token {
  address: Address
  symbol: string
  decimals: number
  active: boolean
}

export interface Quote {
  tokenIn: Address
  tokenOut: Address
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
  slippage: number
  /** Current protocol fee in basis points (read from contract) */
  feeBps: number
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
