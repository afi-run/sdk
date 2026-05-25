/**
 * Example 1: List available tokens on Base
 *
 * Use getTokens() to discover which tokens are supported before
 * building any swap. No private key or RPC required for this call.
 */
import { AfiClient } from "@afi-run/sdk"

const client = new AfiClient({
  rpcUrl: "https://rpc.ankr.com/base/YOUR_API_KEY",
  privateKey: "0xYOUR_PRIVATE_KEY",
})

async function main() {
  const tokens = await client.getTokens()

  console.log(`${tokens.length} tokens available on Base:\n`)

  for (const token of tokens) {
    console.log(`  ${token.symbol.padEnd(10)} ${token.address}  (${token.decimals} decimals)`)
  }

  // Find a specific token by symbol
  const usdc = tokens.find((t) => t.symbol === "USDC")
  const weth = tokens.find((t) => t.symbol === "WETH")

  if (!usdc || !weth) {
    throw new Error("USDC or WETH not found in token list")
  }

  console.log("\nReady to swap:")
  console.log(`  USDC → ${usdc.address}`)
  console.log(`  WETH → ${weth.address}`)
}

main().catch(console.error)
