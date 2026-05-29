/**
 * Example 1: List available tokens
 *
 * getTokens() accepts an optional network (default: NETWORK.BASE).
 * No private key needed — read-only operation.
 */
import { AfiClient, NETWORK } from "@afi-run/sdk"

const client = new AfiClient({
  rpcUrl: "https://rpc.ankr.com/base/YOUR_API_KEY",
})

async function main() {
  // List tokens on Base (default)
  const baseTokens = await client.getTokens()
  console.log(`${baseTokens.length} tokens on Base:\n`)
  for (const token of baseTokens) {
    console.log(`  ${token.symbol.padEnd(10)} ${token.address}  (${token.decimals} decimals)`)
  }

  // List tokens on BSC
  const bscTokens = await client.getTokens(NETWORK.BSC)
  console.log(`\n${bscTokens.length} tokens on BSC:\n`)
  for (const token of bscTokens) {
    console.log(`  ${token.symbol.padEnd(10)} ${token.address}  (${token.decimals} decimals)`)
  }

  // Find specific tokens by symbol
  const usdc = baseTokens.find((t) => t.symbol === "USDC")
  const weth = baseTokens.find((t) => t.symbol === "WETH")

  if (!usdc || !weth) throw new Error("USDC or WETH not found")

  console.log("\nReady to swap:")
  console.log(`  USDC → ${usdc.address}`)
  console.log(`  WETH → ${weth.address}`)
}

main().catch(console.error)
