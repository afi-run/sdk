import type { Address } from "./types.js"

export const AFI_ADDRESS: Address = "0xB8cC65321d169D55b93b4402D795701c6B308ce4"
export const BASE_CHAIN_ID = 8453
export const API_BASE_URL = "https://rpc.afi.run"
export const WETH: Address = "0x4200000000000000000000000000000000000006"
export const MULTICALL3_ADDRESS: Address = "0xcA11bde05977b3631167028862bE2a173976CA11"
/** Default percentage added on top of estimated gas for write txs (approve, swap). */
export const DEFAULT_GAS_BUFFER_PERCENT = 15

/**
 * Block explorer base URLs by network. Override entries here at runtime to swap providers,
 * or pass an explicit `explorer` to txUrl/addressUrl.
 */
export const NETWORK_EXPLORERS: Record<string, string> = {
  base:     "https://basescan.org",
  bsc:      "https://bscscan.com",
  arbitrum: "https://arbiscan.io",
  ethereum: "https://etherscan.io",
  unichain: "https://uniscan.xyz",
}

/** Chain ID per network. */
export const NETWORK_CHAIN_IDS: Record<string, number> = {
  base:     8453,
  bsc:      56,
  arbitrum: 42161,
  ethereum: 1,
  unichain: 130,
}

export const AFI_ABI = [
  {
    type: "function",
    name: "swap",
    inputs: [
      { name: "tokenIn", type: "address" },
      { name: "amountIn", type: "uint256" },
      { name: "tokenOut", type: "address" },
      { name: "minOut", type: "uint256" },
      { name: "params", type: "bytes" },
    ],
    outputs: [{ name: "out", type: "uint256" }],
    stateMutability: "nonpayable",
  },
  {
    type: "function",
    name: "feeBps",
    inputs: [],
    outputs: [{ name: "", type: "uint16" }],
    stateMutability: "view",
  },
  {
    type: "event",
    name: "SwapExecuted",
    inputs: [
      { name: "from", type: "address", indexed: true },
      { name: "assetIn", type: "address", indexed: true },
      { name: "amountIn", type: "uint256", indexed: false },
      { name: "assetOut", type: "address", indexed: true },
      { name: "amountOut", type: "uint256", indexed: false },
    ],
  },
  // ─── Custom errors (verified on Basescan) ────────────────────────────────────
  { type: "error", name: "DifferentAssets", inputs: [
    { name: "expected", type: "address" },
    { name: "actual",   type: "address" },
  ] },
  { type: "error", name: "FeeTooHigh", inputs: [{ name: "feeBps", type: "uint16" }] },
  { type: "error", name: "InsufficientFunds", inputs: [{ name: "available", type: "uint256" }] },
  { type: "error", name: "InvalidRouteID", inputs: [{ name: "routeID", type: "uint16" }] },
  { type: "error", name: "NotOperator", inputs: [] },
  { type: "error", name: "OwnableInvalidOwner", inputs: [{ name: "owner", type: "address" }] },
  { type: "error", name: "OwnableUnauthorizedAccount", inputs: [{ name: "account", type: "address" }] },
  { type: "error", name: "ReentrancyGuardReentrantCall", inputs: [] },
  { type: "error", name: "ZeroAddress", inputs: [] },
] as const

export const MULTICALL3_ABI = [
  {
    type: "function",
    name: "aggregate3",
    inputs: [{
      name: "calls",
      type: "tuple[]",
      components: [
        { name: "target",       type: "address" },
        { name: "allowFailure", type: "bool" },
        { name: "callData",     type: "bytes" },
      ],
    }],
    outputs: [{
      name: "returnData",
      type: "tuple[]",
      components: [
        { name: "success",    type: "bool" },
        { name: "returnData", type: "bytes" },
      ],
    }],
    stateMutability: "payable",
  },
] as const

export const ERC20_ABI = [
  {
    type: "function",
    name: "decimals",
    inputs: [],
    outputs: [{ name: "", type: "uint8" }],
    stateMutability: "view",
  },
  {
    type: "function",
    name: "symbol",
    inputs: [],
    outputs: [{ name: "", type: "string" }],
    stateMutability: "view",
  },
  {
    type: "function",
    name: "name",
    inputs: [],
    outputs: [{ name: "", type: "string" }],
    stateMutability: "view",
  },
  {
    type: "function",
    name: "balanceOf",
    inputs: [{ name: "account", type: "address" }],
    outputs: [{ name: "", type: "uint256" }],
    stateMutability: "view",
  },
  {
    type: "function",
    name: "allowance",
    inputs: [
      { name: "owner", type: "address" },
      { name: "spender", type: "address" },
    ],
    outputs: [{ name: "", type: "uint256" }],
    stateMutability: "view",
  },
  {
    type: "function",
    name: "approve",
    inputs: [
      { name: "spender", type: "address" },
      { name: "value", type: "uint256" },
    ],
    outputs: [{ name: "", type: "bool" }],
    stateMutability: "nonpayable",
  },
] as const
