import { BadRequestError, NetworkError, QuoteError, ServerError } from "./errors.js"
import { encodeSteps } from "./encode-steps.js"
import type { Address, Dex, Hex, Hop, Network, Quote, RpcUrlInfo } from "./types.js"

export interface QuoteRequest {
  tokenIn: Address
  tokenOut: Address
  amountIn: string
  slippage: number
  maxHops: number
  network: Network
  priceBase?: string
  dexs?: Dex[]
  blockNumber?: string | number
  rpcUrls?: RpcUrlInfo[]
}

interface QuoterHop {
  tokenIn: string
  tokenOut: string
  amountIn: string
  amountOut: string
  minOut: string
  amountInRaw: string
  amountOutRaw: string
  minOutRaw: string
  tokenInPrice: string
  tokenOutPrice: string
  slippage: number
  type: string
  kind: string
  routeId: number
  weight: number
}

interface QuoterResponseData {
  tokenIn: string
  tokenOut: string
  amountIn: string
  amountOut: string
  minOut: string
  amountInRaw: string
  amountOutRaw: string
  minOutRaw: string
  steps: string
  slippage: number
  path: string[]
  hops: QuoterHop[]
  tokenInPrice: string
  tokenOutPrice: string
  tokenInBasePrice?: string
  tokenOutBasePrice?: string
}

interface QuoterApiResponse {
  status: "success" | "error"
  message?: string
  data?: QuoterResponseData | string
}

export async function fetchQuote(
  params: QuoteRequest,
  feeBps: number,
  quoterUrl: string,
): Promise<Quote> {
  const body: Record<string, unknown> = {
    network:  params.network,
    tokenIn:  params.tokenIn,
    tokenOut: params.tokenOut,
    amountIn: params.amountIn,
    slippage: params.slippage,
    maxHops:  params.maxHops,
    show:     true,
  }

  if (params.rpcUrls && params.rpcUrls.length > 0) {
    body.rpcUrls = params.rpcUrls
  }
  if (params.priceBase !== undefined) {
    body.priceBase = params.priceBase
  }
  if (params.dexs && params.dexs.length > 0) {
    body.dexs = params.dexs
  }
  if (params.blockNumber !== undefined) {
    body.blockNumber = params.blockNumber
  }

  let res: Response
  try {
    res = await fetch(quoterUrl, {
      method:  "POST",
      headers: { "Content-Type": "application/json" },
      body:    JSON.stringify(body),
    })
  } catch (e) {
    throw new NetworkError(`network error: ${(e as Error).message}`)
  }

  if (!res.ok) {
    const msg = `HTTP ${res.status}`
    if (res.status >= 500) throw new ServerError(msg, res.status)
    if (res.status >= 400) throw new BadRequestError(msg, res.status)
    throw new QuoteError(msg)
  }

  const json = (await res.json()) as QuoterApiResponse

  if (json.status !== "success" || !json.data || typeof json.data === "string") {
    const msg =
      json.message ??
      (typeof json.data === "string" ? json.data : undefined) ??
      "unknown error from quoter"
    throw new QuoteError(msg)
  }

  const d = json.data

  if (!d.minOutRaw || d.minOutRaw === "0") {
    throw new QuoteError("received zero minOut — rejected for safety")
  }

  const hops: Hop[] = (d.hops ?? []).map((h) => ({
    tokenIn:       h.tokenIn as Address,
    tokenOut:      h.tokenOut as Address,
    amountIn:      h.amountIn,
    amountOut:     h.amountOut,
    minOut:        h.minOut,
    amountInWei:   BigInt(h.amountInRaw),
    amountOutWei:  BigInt(h.amountOutRaw),
    minOutWei:     BigInt(h.minOutRaw),
    tokenInPrice:  h.tokenInPrice,
    tokenOutPrice: h.tokenOutPrice,
    slippage:      h.slippage,
    type:          h.type,
    kind:          h.kind,
    routeId:       h.routeId,
    weight:        h.weight,
  }))

  const quote: Quote = {
    tokenIn:       d.tokenIn as Address,
    tokenOut:      d.tokenOut as Address,
    amountIn:      d.amountIn,
    amountOut:     d.amountOut,
    minOut:        d.minOut,
    amountInWei:   BigInt(d.amountInRaw),
    amountOutWei:  BigInt(d.amountOutRaw),
    minOutWei:     BigInt(d.minOutRaw),
    steps:         d.steps as Hex,
    path:          d.path as Address[],
    hops,
    slippage:      d.slippage,
    feeBps,
    tokenInPrice:  d.tokenInPrice,
    tokenOutPrice: d.tokenOutPrice,
    createdAt:     Date.now(),
    network:       params.network,
    maxHops:       params.maxHops,
    priceBase:     params.priceBase,
    dexs:          params.dexs,
  }

  if (d.tokenInBasePrice !== undefined)  quote.tokenInBasePrice  = d.tokenInBasePrice
  if (d.tokenOutBasePrice !== undefined) quote.tokenOutBasePrice = d.tokenOutBasePrice

  return quote
}

// ─── Generic afi-rpc HTTP helpers ────────────────────────────────────────────

/**
 * Base URL passed to all helpers in this section is the API root (e.g.
 * `https://rpc.afi.run`) — NOT a specific endpoint. Each helper appends its
 * own path.
 */

interface GenericApiResponse<T = unknown> {
  status: "success" | "error"
  message?: string
  data?: T | string
}

async function postJson<T>(url: string, body: Record<string, unknown>): Promise<T> {
  let res: Response
  try {
    res = await fetch(url, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    })
  } catch (e) {
    throw new NetworkError(`network error: ${(e as Error).message}`)
  }

  if (!res.ok) {
    const msg = `HTTP ${res.status}`
    if (res.status >= 500) throw new ServerError(msg, res.status)
    if (res.status >= 400) throw new BadRequestError(msg, res.status)
    throw new QuoteError(msg)
  }

  const json = (await res.json()) as GenericApiResponse<T>
  if (json.status !== "success" || json.data === undefined || typeof json.data === "string") {
    const msg =
      json.message ??
      (typeof json.data === "string" ? json.data : undefined) ??
      "unknown error from afi-rpc"
    throw new QuoteError(msg)
  }
  return json.data as T
}

// ─── /arbitrage ──────────────────────────────────────────────────────────────

export interface ArbitrageRequest {
  network: Network
  tokenIn: Address
  /** Cycle end token — set equal to `tokenIn` for a self-funded cycle. */
  tokenOut: Address
  amountIn: string
  /** Optional list of intermediate tokens to consider. */
  tokens?: Address[]
  maxHops?: number
  slippage?: number
  dexs?: Dex[]
  rpcUrls?: RpcUrlInfo[]
}

/**
 * RouteQuote is one route returned by /arbitrage, /command "price", or a per-DEX
 * quote — a single-DEX quote for tokenIn → tokenOut with the encoded hop ready
 * to run through Afi.swap. Feed the best one to quoteFromRoute.
 */
export interface RouteQuote {
  network: string
  type?: string
  kind?: string
  tokenIn: Address
  tokenOut: Address
  tokenInPrice?: string
  tokenOutPrice?: string
  amountIn: string
  amountInRaw: string
  amountOut: string
  amountOutRaw: string
  minOut?: string
  minOutRaw: string
  slippage?: number
  weight?: number
  blockNumber?: string
  fee?: number
  routeId: number
  /** Opaque per-hop calldata for the DEX adapter, hex-encoded. */
  stepData: Hex
}

/** routeProfit returns amountOutRaw − amountInRaw, or null if either is unparseable. */
export function routeProfit(r: RouteQuote): bigint | null {
  try {
    return BigInt(r.amountOutRaw) - BigInt(r.amountInRaw)
  } catch {
    return null
  }
}

/**
 * quoteFromRoute hydrates an executable Quote from a RouteQuote: it wraps the
 * route's single hop ({routeId, stepData}) into Afi.swap params via encodeSteps
 * and copies the token/amount fields. The result can go straight to
 * client.executeSwap / client.simulate.
 *
 * `minOutOverride` replaces the route's minOutRaw — handy for a self-funded
 * cycle that should floor output at principal + threshold and revert otherwise.
 */
export function quoteFromRoute(r: RouteQuote, minOutOverride?: bigint): Quote {
  if (r.routeId < 0 || r.routeId > 0xffff) {
    throw new Error(`quoteFromRoute: routeId ${r.routeId} out of uint16 range`)
  }
  const steps = encodeSteps([{ id: r.routeId, data: (r.stepData ?? "0x") as Hex }])
  return {
    tokenIn: r.tokenIn,
    tokenOut: r.tokenOut,
    amountIn: r.amountIn ?? "",
    amountOut: r.amountOut ?? "",
    minOut: r.minOut ?? "",
    amountInWei: BigInt(r.amountInRaw),
    amountOutWei: r.amountOutRaw ? BigInt(r.amountOutRaw) : 0n,
    minOutWei: minOutOverride ?? BigInt(r.minOutRaw),
    steps,
    path: [r.tokenIn, r.tokenOut],
    hops: [],
    slippage: r.slippage ?? 0,
    feeBps: 0,
    tokenInPrice: r.tokenInPrice ?? "",
    tokenOutPrice: r.tokenOutPrice ?? "",
    createdAt: Date.now(),
    network: r.network as Network,
    maxHops: 1,
  }
}

/**
 * findArbitrage hits POST /arbitrage and returns the candidate routes. Each
 * RouteQuote is an executable single-DEX route — feed the most profitable one to
 * quoteFromRoute. For a self-funded cycle set tokenIn === tokenOut.
 */
export async function findArbitrage(
  apiBaseUrl: string,
  req: ArbitrageRequest,
): Promise<RouteQuote[]> {
  return postJson<RouteQuote[]>(`${apiBaseUrl}/arbitrage`, { ...req })
}

// ─── /command  (dispatch table for action-based endpoints) ──────────────────

export interface CommandRequestBase {
  network: Network
  rpcUrls?: RpcUrlInfo[]
}

export interface PathRequest extends CommandRequestBase {
  tokenIn: Address
  tokenOut: Address
  amountIn?: string
  maxHops?: number
}

/** PathQuote is the priced multi-hop route returned by findPath. */
export interface PathQuote {
  network: string
  path: Address[]
  tokenIn: Address
  tokenOut: Address
  amountIn: string
  amountInRaw: string
  amountOut: string
  amountOutRaw: string
  minOut: string
  minOutRaw: string
  tokenInPrice?: string
  tokenOutPrice?: string
  blockNumber?: string
  slippage?: number
  /** Combined hex-encoded params ready for Afi.swap. */
  steps: Hex
  hops: RouteQuote[]
}

export async function findPath(apiBaseUrl: string, req: PathRequest): Promise<PathQuote> {
  return postJson<PathQuote>(`${apiBaseUrl}/command`, { action: "path", ...req })
}

export interface RoutesRequest extends CommandRequestBase {
  tokenIn?: Address
  tokenOut?: Address
  maxHops?: number
}

/** Route is one candidate token path returned by getRoutes. */
export interface Route {
  path: Address[]
}

export async function getRoutes(apiBaseUrl: string, req: RoutesRequest): Promise<Route[]> {
  return postJson<Route[]>(`${apiBaseUrl}/command`, { action: "routes", ...req })
}

export interface PriceQuoteRequest extends CommandRequestBase {
  tokenIn: Address
  tokenOut: Address
  amountIn: string
}

/** priceQuote returns the per-DEX quotes for the pair (same shape as findArbitrage). */
export async function priceQuote(
  apiBaseUrl: string,
  req: PriceQuoteRequest,
): Promise<RouteQuote[]> {
  return postJson<RouteQuote[]>(`${apiBaseUrl}/command`, { action: "price", ...req })
}

/** Per-DEX quote action names dispatched via `/command`. */
export type DexAction =
  | "uniV3"
  | "cakeV3"
  | "uniV4"
  | "aerodrome"
  | "balancer"
  | "fluid"
  | "curve128"
  | "curve256"

export interface DexQuoteRequest extends CommandRequestBase {
  tokenIn: Address
  tokenOut: Address
  amountIn: string
}

export async function quoteDex(
  apiBaseUrl: string,
  dex: DexAction,
  req: DexQuoteRequest,
): Promise<RouteQuote[]> {
  return postJson<RouteQuote[]>(`${apiBaseUrl}/command`, { action: dex, ...req })
}

// ─── /aave + /liquidation-call ───────────────────────────────────────────────

export interface LiquidationCandidatesRequest {
  network: Network
  skip?: number
  first?: number
  currentTotalDebt?: number
  rpcUrls?: RpcUrlInfo[]
}

export interface AaveCollateral {
  token: string
  tokenAddress: string
  aToken: string
  balanceRaw: string
  balance: string
  decimals: number
}

export interface AavePosition {
  user: string
  debtToken: string
  debtTokenAddress: string
  debtAToken: string
  decimals: number
  debtAmountRaw: string
  debtAmount: string
  collaterals: AaveCollateral[]
}

export async function getLiquidationCandidates(
  apiBaseUrl: string,
  req: LiquidationCandidatesRequest,
): Promise<AavePosition[]> {
  return postJson<AavePosition[]>(`${apiBaseUrl}/aave`, { ...req })
}

export interface LiquidateRequest {
  network: Network
  pool: Address
  user: Address
  tokenIn: Address
  tokenOut: Address
  amountIn: string
  slippage?: number
  rpcUrls?: RpcUrlInfo[]
}

/** LiquidationResult is the executable repay+swap route returned by liquidate. */
export interface LiquidationResult {
  tokenIn: Address
  tokenOut: Address
  amountIn: string
  amountOut: string
  profit: string
  blockNumber?: unknown
  slippage?: number
  /** Combined hex-encoded params ready for Afi.swap. */
  steps: Hex
  hops: RouteQuote[]
}

export async function liquidate(
  apiBaseUrl: string,
  req: LiquidateRequest,
): Promise<LiquidationResult> {
  return postJson<LiquidationResult>(`${apiBaseUrl}/liquidation-call`, { ...req })
}
