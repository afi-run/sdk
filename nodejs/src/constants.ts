import type { Address } from "./types.js"

export const AFI_ADDRESS: Address = "0xB8cC65321d169D55b93b4402D795701c6B308ce4"
export const BASE_CHAIN_ID = 8453
export const QUOTER_URL = "https://rpc.afi.run/quoter"
export const WETH: Address = "0x4200000000000000000000000000000000000006"

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
