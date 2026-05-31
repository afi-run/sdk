import type { Address } from "./types.js"

/**
 * Storage slot index where each ERC-20 stores its `balances` mapping head.
 * For state_override-based simulations, the actual slot for `addr` is
 * `keccak256(addr || slot)`.
 *
 * Slots vary by token:
 *   - Most OZ-based ERC20s: slot 0
 *   - WETH (Base, Mainnet): slot 3
 *   - USDC proxies use slot 9 in their implementation
 *
 * Add entries when popular tokens are missing. For unknown tokens, callers
 * may brute-force detect via `detectBalanceSlot` (slow, in-memory cached).
 */
const BALANCE_SLOTS: Record<number, Record<Address, number>> = {
  8453: {
    // Base
    "0x4200000000000000000000000000000000000006": 3, // WETH
    "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913": 9, // USDC
    "0xd9aAEc86B65D86f6A7B5B1b0c42FFA531710b6CA": 9, // USDbC
    "0x50c5725949A6F0c72E6C4a641F24049A917DB0Cb": 0, // DAI
    "0xfde4C96c8593536E31F229EA8f37b2ADa2699bb2": 0, // USDT
    "0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf": 0, // cbBTC
    "0x2Ae3F1Ec7F1F5012CFEab0185bfc7aa3cf0DEc22": 0, // cbETH
    "0x940181a94A35A4569E4529A3CDfB74e38FD98631": 0, // AERO
    "0x1cea84203673764244e05693e42e6Ace62bE9bA5": 0, // WBTC
    "0x63706e401c06ac8513145b7687A14804d17f814b": 0, // AAVE
    "0x04C0599Ae5A44757c0af6F9eC3b93da8976c150A": 0, // weETH
    "0x60a3E35Cc302bFA44Cb288Bc5a4F316Fdb1adb42": 0, // EURC
  },
  42161: {
    // Arbitrum
    "0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": 51, // WETH
    "0xaf88d065e77c8cC2239327C5EDb3A432268e5831": 9,  // USDC native
    "0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8": 51, // USDC.e
    "0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9": 51, // USDT
    "0xDA10009cBd5D07dd0CeCc66161FC93D7c9000da1": 2,  // DAI
    "0x2f2a2543B76A4166549F7aaB2e75Bef0aefC5B0f": 0,  // WBTC
    "0x912CE59144191C1204E64559FE8253a0e49E6548": 51, // ARB
    "0xfc5A1A6EB076a2C7aD06eD22C90d7E710E35ad0a": 0,  // GMX
    "0x539bdE0d7Dbd336b79148AA742883198BBF60342": 0,  // MAGIC
    "0x0c880f6761F1af8d9Aa9C466984b80DAb9a8c9e8": 0,  // PENDLE
    "0x35751007a407ca6FEFfE80b3cB397736D2cf4dbe": 0,  // weETH
  },
  1: {
    // Mainnet
    "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2": 3, // WETH
    "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48": 9, // USDC
    "0xdAC17F958D2ee523a2206206994597C13D831ec7": 2, // USDT
    "0x6B175474E89094C44Da98b954EedeAC495271d0F": 2, // DAI
    "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599": 0, // WBTC
    "0xae7ab96520DE3A18E5e111B5EaAb095312D7fE84": 0, // stETH
    "0x6c3ea9036406852006290770BEdFcAbA0e23A0e8": 0, // PYUSD
    "0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0": 0, // wstETH
    "0xae78736Cd615f374D3085123A210448E74Fc6393": 1, // rETH
    "0xD533a949740bb3306d119CC777fa900bA034cd52": 3, // CRV
  },
  56: {
    // BSC
    "0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c": 1, // WBNB
    "0x55d398326f99059fF775485246999027B3197955": 1, // USDT
    "0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d": 1, // USDC
    "0xe9e7CEA3DedcA5984780Bafc599bD69ADd087D56": 1, // BUSD
    "0x1AF3F329e8BE154074D8769D1FFa4eE058B1DBc3": 1, // DAI
    "0x2170Ed0880ac9A755fd29B2688956BD959F933F8": 1, // ETH
    "0x7130d2A12B9BCbFAe4f2634d864A1Ee1Ce3Ead9c": 1, // BTCB
    "0x0E09FaBB73Bd3Ade0a17ECC321fD13a19e81cE82": 0, // CAKE
  },
}

const norm = (a: Address): Address => a.toLowerCase() as Address

/** Returns the balance storage slot for the token, or undefined if unknown. */
export function lookupBalanceSlot(chainId: number, token: Address): number | undefined {
  const m = BALANCE_SLOTS[chainId]
  if (!m) return undefined
  // Case-insensitive lookup
  const target = norm(token)
  for (const [k, v] of Object.entries(m)) {
    if (norm(k as Address) === target) return v
  }
  return undefined
}

/** Adds a slot mapping at runtime (e.g., after brute-force detection). */
export function registerBalanceSlot(chainId: number, token: Address, slot: number): void {
  if (!BALANCE_SLOTS[chainId]) BALANCE_SLOTS[chainId] = {}
  BALANCE_SLOTS[chainId][token] = slot
}
