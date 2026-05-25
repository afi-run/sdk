/**
 * Example 4: Full flow using swap() convenience method
 *
 * swap() is a shorthand that calls getQuote() + executeSwap() in sequence.
 * Use it when you don't need to inspect the quote before executing,
 * e.g. in automated bots or scripts.
 *
 * For user-facing apps, prefer the quote → review → executeSwap() pattern
 * shown in example 3.
 */
import { AfiClient } from "@afi-run/sdk"

const USDC = "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
const WETH = "0x4200000000000000000000000000000000000006"

const client = new AfiClient({
  rpcUrl: "https://rpc.ankr.com/base/YOUR_API_KEY",
  privateKey: "0xYOUR_PRIVATE_KEY",
})

async function main() {
  const result = await client.swap({
    tokenIn: USDC,
    tokenOut: WETH,
    amountIn: 500_000000n, // 500 USDC
    slippage: 1.0,         // 1%
  })

  console.log("Swap completed!")
  console.log(`  Tx:         ${result.txHash}`)
  console.log(`  Amount in:  ${result.amountIn} wei USDC`)
  console.log(`  Amount out: ${result.amountOut} wei WETH`)
}

main().catch(console.error)
