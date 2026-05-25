import { QUOTER_URL } from "./constants.js"
import { QuoteError } from "./errors.js"
import type { Address, Hex, Quote, SwapParams } from "./types.js"
import { formatAmount } from "./utils.js"

interface QuoterApiResponse {
  status: "success" | "error"
  message?: string
  data?: {
    tokenIn: string
    tokenOut: string
    amountInRaw: string
    amountOutRaw: string
    minOutRaw: string
    steps: string
    slippage: number
    path: string[]
  }
}

export async function fetchQuote(
  params: SwapParams,
  decimals: number,
  feeBps: number,
  rpcUrl: string,
): Promise<Quote> {
  const body = {
    network: "base",
    tokenIn: params.tokenIn,
    tokenOut: params.tokenOut,
    amountIn: formatAmount(params.amountIn, decimals),
    slippage: params.slippage,
    maxHops: 4,
    show: true,
    rpcUrl,
  }

  let res: Response
  try {
    res = await fetch(QUOTER_URL, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    })
  } catch (e) {
    throw new QuoteError(`network error: ${(e as Error).message}`)
  }

  if (!res.ok) {
    throw new QuoteError(`HTTP ${res.status}`)
  }

  const json = (await res.json()) as QuoterApiResponse

  if (json.status !== "success" || !json.data) {
    throw new QuoteError(json.message ?? "unknown error from quoter")
  }

  const d = json.data

  if (!d.minOutRaw || d.minOutRaw === "0") {
    throw new QuoteError("received zero minOut — rejected for safety")
  }

  return {
    tokenIn: d.tokenIn as Address,
    tokenOut: d.tokenOut as Address,
    amountInWei: BigInt(d.amountInRaw),
    amountOutWei: BigInt(d.amountOutRaw),
    minOutWei: BigInt(d.minOutRaw),
    steps: d.steps as Hex,
    path: d.path as Address[],
    slippage: d.slippage,
    feeBps,
  }
}
