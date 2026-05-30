import type { PublicClient, Transport } from "viem"
import type { base } from "viem/chains"
import { AFI_ADDRESS, ERC20_ABI, MULTICALL3_ADDRESS } from "./constants.js"
import type { Address, TokenInfo } from "./types.js"

type BasePublicClient = PublicClient<Transport, typeof base>

const ZERO_ADDRESS = "0x0000000000000000000000000000000000000000" as Address

function emptyOwner(owner: Address | undefined): boolean {
  return !owner || owner.toLowerCase() === ZERO_ADDRESS
}

/** Immutable per-token metadata that the cache can safely reuse forever. */
export interface TokenMetadata {
  symbol:   string
  name:     string
  decimals: number
}

/** Public type for a single contract call passed to `client.multicall()`. */
export interface MulticallContract {
  address:      Address
  abi:          readonly unknown[]
  functionName: string
  args?:        readonly unknown[]
}

/** Public type for a single multicall result. */
export type MulticallResult =
  | { status: "success"; result: unknown }
  | { status: "failure"; error: Error }

/**
 * Generic multicall wrapper. Bundles N read calls into a single RPC round-trip
 * via Multicall3. Use for arbitrary contract reads beyond what the SDK exposes.
 */
export async function genericMulticall(
  client: BasePublicClient,
  contracts: MulticallContract[],
  opts: { allowFailure?: boolean } = {},
): Promise<MulticallResult[]> {
  const allowFailure = opts.allowFailure ?? true
  const results = (await client.multicall({
    contracts: contracts as any,
    multicallAddress: MULTICALL3_ADDRESS,
    allowFailure,
  })) as Array<{ status: "success"; result: unknown } | { status: "failure"; error?: Error }>
  return results.map((r) =>
    r.status === "success"
      ? { status: "success", result: r.result }
      : { status: "failure", error: r.error ?? new Error("call reverted") },
  )
}

/**
 * Fetches token metadata in a single multicall.
 * When `owner` is provided, also includes balance + allowance against the AFI contract.
 *
 * `metaCache` (optional) caches immutable metadata across calls — pass the client's
 * shared cache to avoid re-fetching symbol/name/decimals for repeated lookups.
 */
export async function fetchTokenInfo(
  token: Address,
  client: BasePublicClient,
  owner?: Address,
  metaCache?: Map<string, TokenMetadata>,
): Promise<TokenInfo> {
  const [info] = await fetchTokenInfoBatch([token], client, owner, metaCache)
  return info
}

/**
 * Batched form of fetchTokenInfo. Returns results in the same order as the input.
 * When `metaCache` is provided, tokens with cached metadata only fetch
 * balance/allowance (or nothing, if no owner is given).
 */
export async function fetchTokenInfoBatch(
  tokens: Address[],
  client: BasePublicClient,
  owner?: Address,
  metaCache?: Map<string, TokenMetadata>,
): Promise<TokenInfo[]> {
  if (tokens.length === 0) return []
  const withOwner = !emptyOwner(owner)

  // Partition tokens by whether their metadata is already cached.
  type Plan = { token: Address; cached: TokenMetadata | undefined; offset: number; count: number }
  const plans: Plan[] = []
  const contracts: MulticallContract[] = []
  for (const token of tokens) {
    const cached = metaCache?.get(token.toLowerCase())
    const plan: Plan = { token, cached, offset: contracts.length, count: 0 }
    if (!cached) {
      contracts.push(
        { address: token, abi: ERC20_ABI, functionName: "symbol" },
        { address: token, abi: ERC20_ABI, functionName: "name" },
        { address: token, abi: ERC20_ABI, functionName: "decimals" },
      )
      plan.count += 3
    }
    if (withOwner) {
      contracts.push(
        { address: token, abi: ERC20_ABI, functionName: "balanceOf", args: [owner!] },
        { address: token, abi: ERC20_ABI, functionName: "allowance", args: [owner!, AFI_ADDRESS] },
      )
      plan.count += 2
    }
    plans.push(plan)
  }

  // If nothing to fetch (everything cached and no owner), return purely from cache.
  const results: MulticallResult[] =
    contracts.length > 0
      ? await genericMulticall(client, contracts, { allowFailure: true })
      : []

  return plans.map((plan) => {
    let cursor = plan.offset
    let meta: TokenMetadata
    if (plan.cached) {
      meta = plan.cached
    } else {
      const symRes = results[cursor++]
      const nameRes = results[cursor++]
      const decRes = results[cursor++]
      if (decRes.status !== "success") throw new Error(`decimals() reverted for ${plan.token}`)
      meta = {
        symbol:   symRes.status === "success" ? (symRes.result as string) : "",
        name:     nameRes.status === "success" ? (nameRes.result as string) : "",
        decimals: Number(decRes.result as number | bigint),
      }
      metaCache?.set(plan.token.toLowerCase(), meta)
    }

    const info: TokenInfo = { address: plan.token, ...meta }
    if (withOwner) {
      const balRes = results[cursor++]
      const allowRes = results[cursor++]
      if (balRes.status !== "success") throw new Error(`balanceOf() reverted for ${plan.token}`)
      if (allowRes.status !== "success") throw new Error(`allowance() reverted for ${plan.token}`)
      info.owner = owner
      info.balance = balRes.result as unknown as bigint
      info.allowance = allowRes.result as unknown as bigint
    }
    return info
  })
}
