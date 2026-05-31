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
import { lookupBalanceSlot } from "./token-slots.js"

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
 * Common use cases:
 *   - Validate arbitrage profitability before flashloan submission
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

  const slot = lookupBalanceSlot(opts.chainId, opts.asset)
  if (slot === undefined) {
    throw new Error(
      `simulateRoute: no static balance slot for token ${opts.asset} on chain ${opts.chainId} ` +
        `(call detectBalanceSlot to seed)`,
    )
  }

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
