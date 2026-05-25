/**
 * Example 2: Get a quote and inspect it before deciding to swap
 *
 * getQuote() only reads data — it does NOT send any transaction.
 * Use it to show pricing to the user before they confirm the swap.
 */
import { AfiClient } from "@afi-run/sdk"

const USDC = "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
const WETH = "0x4200000000000000000000000000000000000006"

const client = new AfiClient({
  rpcUrl: "https://rpc.ankr.com/base/YOUR_API_KEY",
  privateKey: "0xYOUR_PRIVATE_KEY",
})

async function main() {
  const amountIn = 1000_000000n // 1000 USDC (6 decimals)

  console.log("Fetching quote for 1000 USDC → WETH...\n")

  const quote = await client.getQuote({
    tokenIn: USDC,
    tokenOut: WETH,
    amountIn,
    slippage: 0.5, // 0.5%
  })

  // All amounts are in raw wei (bigint)
  // Divide by 10^decimals to get human-readable values

  const amountOutEth = Number(quote.amountOutWei) / 1e18
  const minOutEth = Number(quote.minOutWei) / 1e18
  const feePercent = (quote.feeBps / 100).toFixed(2)

  console.log("Quote details:")
  console.log(`  You send:       1000 USDC`)
  console.log(`  You receive:    ~${amountOutEth.toFixed(6)} WETH`)
  console.log(`  Minimum out:    ${minOutEth.toFixed(6)} WETH  (with ${quote.slippage}% slippage)`)
  console.log(`  Protocol fee:   ${feePercent}%  (${quote.feeBps} bps, read live from contract)`)
  console.log(`  Route:          ${quote.path.join(" → ")}`)
  console.log(`  Hops:           ${quote.path.length - 1}`)

  // The quote object is self-contained — pass it directly to executeSwap()
  // when the user confirms. No need to re-fetch.
  console.log("\nQuote is ready. Pass it to executeSwap() to proceed.")
  return quote
}

main().catch(console.error)
