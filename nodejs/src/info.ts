import type { Token } from "./types.js"

const INFO_URL = "https://rpc.afi.run/info"

interface InfoResponse {
  status: "success" | "error"
  data?: {
    base?: RawToken[]
    [network: string]: RawToken[] | undefined
  }
}

interface RawToken {
  address: string
  symbol: string
  decimals: number
  active: boolean
}

export async function fetchTokens(): Promise<Token[]> {
  let res: Response
  try {
    res = await fetch(INFO_URL)
  } catch (e) {
    throw new Error(`AFI info request failed: ${(e as Error).message}`)
  }

  if (!res.ok) throw new Error(`AFI info HTTP ${res.status}`)

  const json = (await res.json()) as InfoResponse

  if (json.status !== "success" || !json.data?.base) {
    throw new Error("AFI info returned no data for base network")
  }

  return json.data.base.map((t) => ({
    address: t.address as `0x${string}`,
    symbol: t.symbol,
    decimals: t.decimals,
    active: t.active,
  }))
}
