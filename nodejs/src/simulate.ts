import {
  decodeAbiParameters,
  encodeAbiParameters,
  encodeFunctionData,
  keccak256,
  pad,
  type Address,
  type Hex,
  type PublicClient,
} from "viem"
import { lookupBalanceSlot, registerBalanceSlot } from "./token-slots.js"

/** Function selector + arg types for RouteQuoter.quote */
export const ROUTE_QUOTER_ABI = [
  {
    type: "function",
    name: "quote",
    inputs: [
      { name: "asset", type: "address" },
      { name: "amount", type: "uint256" },
      { name: "params", type: "bytes" },
    ],
    outputs: [
      { name: "outputAsset", type: "address" },
      { name: "amountOut", type: "uint256" },
    ],
    stateMutability: "nonpayable",
  },
] as const

export interface SimulationResult {
  /** Asset at the end of the route chain. For arbitrage cycles, equals input asset. */
  outputAsset?: Address
  /** Amount produced by executing the full chain. */
  amountOut?: bigint
  /** True if any route reverted internally. */
  reverted: boolean
  /** Raw revert payload when reverted=true. */
  revertData?: Hex
}

export interface SimulateOpts {
  /** A viem PublicClient connected to the target chain. */
  publicClient: PublicClient
  /** Chain ID — used to look up the token's balance storage slot. */
  chainId: number
  /** Deployed RouteQuoter contract on this chain. */
  quoterAddress: Address
  /** Input token entering the route chain. */
  asset: Address
  /** Input amount (raw units). */
  amount: bigint
  /** Tight-format step encoding (same as Lib.runRoutes expects). */
  stepsEncoded: Hex
  /** Block number to simulate against. Omit for "latest". */
  blockNumber?: bigint
}

/**
 * Simulates a full route chain via eth_call with state_override, granting the
 * RouteQuoter contract a virtual balance of `amount` of `asset`. Returns the
 * actual output that the chain would produce, or revert details on failure.
 *
 * If `asset`'s balance slot is not in the static table, it is auto-detected
 * on-chain via {@link detectBalanceSlot} and cached (extra eth_calls the first
 * time a new token is seen). This keeps simulation working for any token the
 * backend lists without hand-maintaining the slot table.
 *
 * Common use cases:
 *   - Validate arbitrage-cycle profitability before submitting the swap
 *   - Pre-flight a user swap to detect reverts before signing
 *   - Compare actual output vs DEX view-quote for slippage calibration
 *
 * @example
 * ```ts
 * const sim = await simulateRoute({
 *   publicClient,
 *   chainId: 8453,
 *   quoterAddress: ROUTE_QUOTER_BASE,
 *   asset: USDC,
 *   amount: 1000_000_000n,
 *   stepsEncoded: quote.steps,
 * })
 * if (sim.reverted) throw new Error("would revert")
 * if (sim.amountOut! <= 1000_000_000n) console.log("not profitable")
 * ```
 */
export async function simulateRoute(opts: SimulateOpts): Promise<SimulationResult> {
  if (opts.amount <= 0n) throw new Error("simulateRoute: amount must be > 0")

  const slot =
    lookupBalanceSlot(opts.chainId, opts.asset) ??
    (await detectBalanceSlot(opts.publicClient, opts.chainId, opts.asset))

  const derivedSlot = mappingSlot(opts.quoterAddress, slot)

  const callData = encodeFunctionData({
    abi: ROUTE_QUOTER_ABI,
    functionName: "quote",
    args: [opts.asset, opts.amount, opts.stepsEncoded],
  })

  try {
    const { data } = await opts.publicClient.call({
      to: opts.quoterAddress,
      data: callData,
      stateOverride: [
        {
          address: opts.asset,
          stateDiff: [
            {
              slot: derivedSlot,
              value: pad(`0x${opts.amount.toString(16)}` as Hex, { size: 32 }),
            },
          ],
        },
      ],
      blockNumber: opts.blockNumber,
    })

    if (!data) return { reverted: true }

    const [outputAsset, amountOut] = decodeAbiParameters(
      [
        { name: "outputAsset", type: "address" },
        { name: "amountOut", type: "uint256" },
      ],
      data,
    ) as [Address, bigint]

    return { outputAsset, amountOut, reverted: false }
  } catch (err: any) {
    // viem surfaces revert data in err.cause.data or err.data
    const revertData =
      (err?.cause?.data as Hex | undefined) ??
      (err?.data as Hex | undefined) ??
      undefined
    return { reverted: true, revertData }
  }
}

/**
 * Computes keccak256(holder padded to 32 bytes || slot padded to 32 bytes).
 * This is the storage location of a single mapping(address => *) entry.
 */
export function mappingSlot(holder: Address, slot: number | bigint): Hex {
  const encoded = encodeAbiParameters(
    [{ type: "address" }, { type: "uint256" }],
    [holder, typeof slot === "number" ? BigInt(slot) : slot],
  )
  return keccak256(encoded)
}

const BALANCEOF_ABI = [
  {
    type: "function",
    name: "balanceOf",
    inputs: [{ name: "account", type: "address" }],
    outputs: [{ name: "", type: "uint256" }],
    stateMutability: "view",
  },
] as const

const DETECT_HOLDER = "0x0000000000000000000000000000000000001234" as Address
const DETECT_SENTINEL = 0xdeadbeefdeadbeefn

/**
 * Brute-forces storage slots `0..maxSlot`, overriding each candidate with a
 * sentinel balance and calling `balanceOf` until the reading matches — i.e. the
 * slot that actually backs the token's `balances` mapping. Slow (one eth_call
 * per slot tested) but runs once per token: on success the slot is registered
 * via {@link registerBalanceSlot} so later lookups hit the cache.
 *
 * Mirrors the Go SDK's `DetectBalanceSlot`. Throws if no slot matches within
 * `maxSlot` (default 128; the highest curated slot is 51).
 */
export async function detectBalanceSlot(
  publicClient: PublicClient,
  chainId: number,
  token: Address,
  maxSlot = 128,
): Promise<number> {
  const want = pad(`0x${DETECT_SENTINEL.toString(16)}` as Hex, { size: 32 })
  const data = encodeFunctionData({
    abi: BALANCEOF_ABI,
    functionName: "balanceOf",
    args: [DETECT_HOLDER],
  })

  for (let slot = 0; slot <= maxSlot; slot++) {
    try {
      const { data: out } = await publicClient.call({
        to: token,
        data,
        stateOverride: [
          {
            address: token,
            stateDiff: [{ slot: mappingSlot(DETECT_HOLDER, slot), value: want }],
          },
        ],
      })
      if (out && BigInt(out) === DETECT_SENTINEL) {
        registerBalanceSlot(chainId, token, slot)
        return slot
      }
    } catch {
      // non-standard token / RPC hiccup on this slot — try the next
    }
  }
  throw new Error(
    `detectBalanceSlot: balance slot not detected for ${token} on chain ${chainId} within ${maxSlot} slots`,
  )
}
