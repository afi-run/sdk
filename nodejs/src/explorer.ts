import { NETWORK_EXPLORERS } from "./constants.js"
import type { Network } from "./types.js"
import { NETWORK } from "./types.js"

function baseFor(network: Network, override?: string): string {
  const base = override ?? NETWORK_EXPLORERS[network]
  if (!base) throw new Error(`no explorer URL configured for network "${network}"`)
  return base.replace(/\/+$/, "")
}

/**
 * Returns the explorer URL for a transaction hash on `network` (default: BASE).
 *
 * @example
 * txUrl("0xabc...")                       // https://basescan.org/tx/0xabc...
 * txUrl("0xabc...", NETWORK.BSC)          // https://bscscan.com/tx/0xabc...
 * txUrl(hash, NETWORK.BASE, "https://...")// custom explorer base
 */
export function txUrl(hash: string, network: Network = NETWORK.BASE, explorer?: string): string {
  return `${baseFor(network, explorer)}/tx/${hash}`
}

/**
 * Returns the explorer URL for an address on `network` (default: BASE).
 */
export function addressUrl(address: string, network: Network = NETWORK.BASE, explorer?: string): string {
  return `${baseFor(network, explorer)}/address/${address}`
}
