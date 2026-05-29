import { QuoteError } from "./errors.js"
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
    throw new QuoteError(`network error: ${(e as Error).message}`)
  }

  if (!res.ok) {
    throw new QuoteError(`HTTP ${res.status}`)
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
  }

  if (d.tokenInBasePrice !== undefined)  quote.tokenInBasePrice  = d.tokenInBasePrice
  if (d.tokenOutBasePrice !== undefined) quote.tokenOutBasePrice = d.tokenOutBasePrice

  return quote
}
