import type { Address } from "./types.js"

// Deployed 2026-05-30. Re-populate after redeploy.

/**
 * Deprecated single-chain default. Use AFI_ADDRESSES[chainId] for multi-chain
 * deployments.
 */
export const AFI_ADDRESS: Address = "0xFd4F8822f13D01aB142Bc985Ce587E35d7673C6e"

/**
 * Deployed Afi router per chain ID.
 */
export const AFI_ADDRESSES: Record<number, Address> = {
  1:     "0xc578a4e89795803F396160610F4990c44abA8dAb", // Ethereum
  56:    "0xFd4F8822f13D01aB142Bc985Ce587E35d7673C6e", // BSC
  130:   "0xFd4F8822f13D01aB142Bc985Ce587E35d7673C6e", // Unichain
  8453:  "0xFd4F8822f13D01aB142Bc985Ce587E35d7673C6e", // Base
  42161: "0xd74F60BD38243d089e286E3B6b9348f43a2314dF", // Arbitrum
}

/**
 * Deployed RouteQuoter per chain ID (used by simulateRoute).
 */
export const ROUTE_QUOTER_ADDRESSES: Record<number, Address> = {
  1:     "0x5e41b417E9742DB9c5402F8B1969a33891628Bed", // Ethereum
  56:    "0xcA37E05a20E93fD88E5367F9d7d1422937c57A38", // BSC
  130:   "0x2Cc852Cd57CC1b57CA09dbA7f69F0e225008cEBE", // Unichain
  8453:  "0xB5637138Cee6e757B679FFF8aDEA8DBa3E7544bB", // Base
  42161: "0xBdD42B4fF06aCa8908D5E5d4826fFf5cdaC43895", // Arbitrum
}

/**
 * Deployed NMR (Nathan flash-loan executor) per chain ID. Only chains with
 * Aave V3 receive a deployment.
 */
export const NMR_ADDRESSES: Record<number, Address> = {
  1:     "0x29EfbFC1534A9B7af02142A5D97454E24Dc51b3a", // Ethereum
  8453:  "0xefA12ba0196FD5ec44AF2ecAddc17333dF5FA779", // Base
  42161: "0x6b533D53ec93eC30963b38576Ed8330Ff346a723", // Arbitrum
}

export const BASE_CHAIN_ID = 8453
export const API_BASE_URL = "https://rpc.afi.run"
export const WETH: Address = "0x4200000000000000000000000000000000000006"
export const MULTICALL3_ADDRESS: Address = "0xcA11bde05977b3631167028862bE2a173976CA11"
/** Default percentage added on top of estimated gas for write txs (approve, swap). */
export const DEFAULT_GAS_BUFFER_PERCENT = 15

/** Mirrors NMR.MAX_PROFIT_SHARE — operator profit cap in percent. */
export const MAX_PROFIT_SHARE = 50

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
    name: "swapFor",
    inputs: [
      { name: "user", type: "address" },
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
    name: "batchSwapFor",
    inputs: [
      {
        name: "swaps",
        type: "tuple[]",
        components: [
          { name: "user", type: "address" },
          { name: "tokenIn", type: "address" },
          { name: "amountIn", type: "uint256" },
          { name: "tokenOut", type: "address" },
          { name: "minOut", type: "uint256" },
          { name: "params", type: "bytes" },
        ],
      },
    ],
    outputs: [],
    stateMutability: "nonpayable",
  },
  {
    type: "function",
    name: "paused",
    inputs: [],
    outputs: [{ name: "", type: "bool" }],
    stateMutability: "view",
  },
  {
    type: "function",
    name: "feeBps",
    inputs: [],
    outputs: [{ name: "", type: "uint16" }],
    stateMutability: "view",
  },
  {
    type: "function",
    name: "feeBpsOf",
    inputs: [{ name: "user", type: "address" }],
    outputs: [{ name: "", type: "uint16" }],
    stateMutability: "view",
  },
  {
    type: "function",
    name: "treasury",
    inputs: [],
    outputs: [{ name: "", type: "address" }],
    stateMutability: "view",
  },
  {
    type: "function",
    name: "registry",
    inputs: [],
    outputs: [{ name: "", type: "address" }],
    stateMutability: "view",
  },
  {
    type: "function",
    name: "hasRules",
    inputs: [],
    outputs: [{ name: "", type: "bool" }],
    stateMutability: "view",
  },
  {
    type: "function",
    name: "primaryOperator",
    inputs: [],
    outputs: [{ name: "", type: "address" }],
    stateMutability: "view",
  },
  {
    type: "function",
    name: "isOperator",
    inputs: [{ name: "account", type: "address" }],
    outputs: [{ name: "", type: "bool" }],
    stateMutability: "view",
  },
  {
    type: "function",
    name: "owner",
    inputs: [],
    outputs: [{ name: "", type: "address" }],
    stateMutability: "view",
  },
  {
    type: "function",
    name: "pendingOwner",
    inputs: [],
    outputs: [{ name: "", type: "address" }],
    stateMutability: "view",
  },
  {
    type: "function",
    name: "MAX_FEE_BPS",
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
  {
    type: "event",
    name: "FeeCollected",
    inputs: [
      { name: "token", type: "address", indexed: true },
      { name: "amount", type: "uint256", indexed: false },
    ],
  },
  {
    type: "event",
    name: "TreasuryUpdated",
    inputs: [
      { name: "treasury", type: "address", indexed: true },
    ],
  },
  {
    type: "event",
    name: "FeeBpsUpdated",
    inputs: [
      { name: "feeBps", type: "uint16", indexed: false },
    ],
  },
  {
    type: "event",
    name: "UserFeeBpsSet",
    inputs: [
      { name: "user", type: "address", indexed: true },
      { name: "feeBps", type: "uint16", indexed: false },
    ],
  },
  {
    type: "event",
    name: "UserFeeBpsCleared",
    inputs: [
      { name: "user", type: "address", indexed: true },
    ],
  },
  {
    type: "event",
    name: "AnyUserOverrideReset",
    inputs: [],
  },
  // ─── Owner-only admin functions ──────────────────────────────────────────────
  {
    type: "function",
    name: "pause",
    inputs: [],
    outputs: [],
    stateMutability: "nonpayable",
  },
  {
    type: "function",
    name: "unpause",
    inputs: [],
    outputs: [],
    stateMutability: "nonpayable",
  },
  {
    type: "function",
    name: "setTreasury",
    inputs: [{ name: "_treasury", type: "address" }],
    outputs: [],
    stateMutability: "nonpayable",
  },
  {
    type: "function",
    name: "setFeeBps",
    inputs: [{ name: "_feeBps", type: "uint16" }],
    outputs: [],
    stateMutability: "nonpayable",
  },
  {
    type: "function",
    name: "setUserFeeBps",
    inputs: [
      { name: "user", type: "address" },
      { name: "_feeBps", type: "uint16" },
    ],
    outputs: [],
    stateMutability: "nonpayable",
  },
  {
    type: "function",
    name: "setUserFeeBpsBatch",
    inputs: [
      { name: "users", type: "address[]" },
      { name: "feesBps", type: "uint16[]" },
    ],
    outputs: [],
    stateMutability: "nonpayable",
  },
  {
    type: "function",
    name: "clearUserFeeBps",
    inputs: [{ name: "user", type: "address" }],
    outputs: [],
    stateMutability: "nonpayable",
  },
  {
    type: "function",
    name: "resetAnyUserOverride",
    inputs: [],
    outputs: [],
    stateMutability: "nonpayable",
  },
  {
    type: "function",
    name: "addRule",
    inputs: [{ name: "_rule", type: "address" }],
    outputs: [],
    stateMutability: "nonpayable",
  },
  {
    type: "function",
    name: "clearRules",
    inputs: [],
    outputs: [],
    stateMutability: "nonpayable",
  },
  {
    type: "function",
    name: "setOperator",
    inputs: [
      { name: "_operator", type: "address" },
      { name: "_value", type: "bool" },
    ],
    outputs: [],
    stateMutability: "nonpayable",
  },
  {
    type: "function",
    name: "rescueTokens",
    inputs: [
      { name: "token", type: "address" },
      { name: "value", type: "uint256" },
      { name: "to", type: "address" },
    ],
    outputs: [],
    stateMutability: "nonpayable",
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

export const NMR_ABI = [
  {
    type: "function",
    name: "requestOperation",
    inputs: [
      { name: "asset", type: "address" },
      { name: "amount", type: "uint256" },
      { name: "params", type: "bytes" },
    ],
    outputs: [],
    stateMutability: "nonpayable",
  },
  {
    type: "function",
    name: "swap",
    inputs: [
      { name: "tokenIn", type: "address" },
      { name: "amountIn", type: "uint256" },
      { name: "minOut", type: "uint256" },
      { name: "params", type: "bytes" },
    ],
    outputs: [{ name: "out", type: "uint256" }],
    stateMutability: "nonpayable",
  },
  {
    type: "function",
    name: "loan",
    inputs: [
      { name: "user",   type: "address" },
      { name: "asset",  type: "address" },
      { name: "amount", type: "uint256" },
      { name: "minOut", type: "uint256" },
      { name: "params", type: "bytes"   },
    ],
    outputs: [{ name: "", type: "uint256" }],
    stateMutability: "nonpayable",
  },
  {
    type: "function",
    name: "sweepProfit",
    inputs: [
      { name: "token", type: "address" },
      { name: "amount", type: "uint256" },
    ],
    outputs: [],
    stateMutability: "nonpayable",
  },
  {
    type: "function",
    name: "setTreasury",
    inputs: [{ name: "treasury", type: "address" }],
    outputs: [],
    stateMutability: "nonpayable",
  },
  {
    type: "function",
    name: "treasury",
    inputs: [],
    outputs: [{ name: "", type: "address" }],
    stateMutability: "view",
  },
  {
    type: "function",
    name: "PROFIT_SHARE",
    inputs: [],
    outputs: [{ name: "", type: "uint8" }],
    stateMutability: "view",
  },
  {
    type: "function",
    name: "isOperator",
    inputs: [{ name: "account", type: "address" }],
    outputs: [{ name: "", type: "bool" }],
    stateMutability: "view",
  },
  {
    type: "event",
    name: "FlashLoanRequested",
    inputs: [
      { name: "asset", type: "address", indexed: true },
      { name: "amount", type: "uint256", indexed: false },
    ],
  },
  {
    type: "event",
    name: "FlashLoanExecuted",
    inputs: [
      { name: "asset", type: "address", indexed: true },
      { name: "amount", type: "uint256", indexed: false },
      { name: "premium", type: "uint256", indexed: false },
      { name: "profit", type: "uint256", indexed: false },
    ],
  },
  {
    type: "event",
    name: "FlashLoanFailed",
    inputs: [
      { name: "asset", type: "address", indexed: true },
      { name: "amount", type: "uint256", indexed: false },
      { name: "reason", type: "string", indexed: false },
    ],
  },
  {
    type: "event",
    name: "FlashLoanFailedWithData",
    inputs: [
      { name: "asset", type: "address", indexed: true },
      { name: "amount", type: "uint256", indexed: false },
      { name: "data", type: "bytes", indexed: false },
    ],
  },
  {
    type: "event",
    name: "SwapExecuted",
    inputs: [
      { name: "assetIn", type: "address", indexed: true },
      { name: "amountIn", type: "uint256", indexed: false },
      { name: "assetOut", type: "address", indexed: true },
      { name: "amountOut", type: "uint256", indexed: false },
    ],
  },
  {
    type: "event",
    name: "ProfitSwept",
    inputs: [
      { name: "asset", type: "address", indexed: true },
      { name: "amount", type: "uint256", indexed: false },
      { name: "to", type: "address", indexed: true },
    ],
  },
  {
    type: "event",
    name: "TreasuryUpdated",
    inputs: [{ name: "treasury", type: "address", indexed: true }],
  },
  {
    type: "event",
    name: "ProfitShareUpdated",
    inputs: [{ name: "profitShare", type: "uint8", indexed: false }],
  },
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

/** Minimal RouteRegistry ABI — used by AfiClient.getRoute / listRoutes. */
export const ROUTE_REGISTRY_ABI = [
  {
    type: "function",
    name: "getRoute",
    inputs: [{ name: "id", type: "uint16" }],
    outputs: [{ name: "", type: "address" }],
    stateMutability: "view",
  },
  {
    type: "function",
    name: "getRoutes",
    inputs: [{ name: "ids", type: "uint16[]" }],
    outputs: [{ name: "addrs", type: "address[]" }],
    stateMutability: "view",
  },
] as const

/**
 * Minimal Ownable2Step ABI — used by AfiClient.acceptOwnership and friends.
 * Mirrors OpenZeppelin's Ownable2Step surface needed by the SDK.
 */
export const OWNABLE2STEP_ABI = [
  {
    type: "function",
    name: "acceptOwnership",
    inputs: [],
    outputs: [],
    stateMutability: "nonpayable",
  },
  {
    type: "function",
    name: "pendingOwner",
    inputs: [],
    outputs: [{ name: "", type: "address" }],
    stateMutability: "view",
  },
  {
    type: "function",
    name: "owner",
    inputs: [],
    outputs: [{ name: "", type: "address" }],
    stateMutability: "view",
  },
] as const

/**
 * Expected treasury address for the Afi router. `verifyDeployment` warns when
 * the on-chain `treasury()` slot drifts from this value. Leave as the zero
 * address to disable the check (e.g. while the operator is still rotating).
 */
export const TREASURY_OWNER: Address = "0x0000000000000000000000000000000000000000"

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
