import type { Network, Token } from "./types.js"

interface RawToken {
  address: string
  symbol: string
  decimals: number
  active: boolean
}

interface InfoResponse {
  status: "success" | "error"
  data?: Record<string, RawToken[] | undefined>
}

export async function fetchTokens(network: Network, infoUrl: string): Promise<Token[]> {
  return fetchTokensFrom(network, infoUrl)
}

export async function fetchTokensFrom(network: Network, url: string): Promise<Token[]> {
  const fullUrl = `${url}?network=${network}`

  let res: Response
  try {
    res = await fetch(fullUrl)
  } catch (e) {
    throw new Error(`AFI info request failed: ${(e as Error).message}`)
  }

  if (!res.ok) throw new Error(`AFI info HTTP ${res.status}`)

  const json = (await res.json()) as InfoResponse

  if (json.status !== "success" || !json.data) {
    throw new Error(`AFI info returned no data for network: ${network}`)
  }

  const raw = json.data[network]
  if (!raw || raw.length === 0) {
    throw new Error(`AFI info returned no tokens for network: ${network}`)
  }

  return raw.map((t) => ({
    address:  t.address as `0x${string}`,
    symbol:   t.symbol,
    decimals: t.decimals,
    active:   t.active,
  }))
}
