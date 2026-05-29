/**
 * Example 4: One-call convenience flow using .execute()
 *
 * QuoteBuilder.execute() fetches the quote and runs executeSwap() in one call.
 * Use for bots and scripts where you don't need to review the quote first.
 *
 * For user-facing apps, prefer example 3 (quote → review → execute).
 */
import { AfiClient, NETWORK, DEX, formatUnits } from "@afi-run/sdk"

const USDC = "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
const WETH = "0x4200000000000000000000000000000000000006"

const client = new AfiClient({
  rpcUrl: "https://rpc.ankr.com/base/YOUR_API_KEY",
}).connect("0xYOUR_PRIVATE_KEY")

async function main() {
  // Variation A: minimal — just the essentials
  const result = await client
    .quote(USDC, WETH, "500")
    .slippage(1.0)
    .execute()

  console.log("Swap completed!")
  console.log(`  Tx:         ${result.txHash}`)
  console.log(`  Amount in:  ${formatUnits(result.amountIn, 6)} USDC`)
  console.log(`  Amount out: ${formatUnits(result.amountOut, 18)} WETH`)

  // Variation B: with optional parameters
  // const result2 = await client
  //   .quote(USDC, WETH, "200")
  //   .slippage(0.5)
  //   .maxHops(3)
  //   .network(NETWORK.BASE)
  //   .dexs(DEX.UNI_V3, DEX.AERODROME)
  //   .execute()
}

main().catch(console.error)
