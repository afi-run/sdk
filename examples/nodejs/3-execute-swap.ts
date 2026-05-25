/**
 * Example 3: Quote → review → execute (recommended flow)
 *
 * This is the recommended pattern for production applications:
 * 1. Fetch a quote (no tx sent)
 * 2. Show pricing to the user
 * 3. User confirms
 * 4. Call executeSwap(quote) — this handles approve + simulate + swap
 *
 * executeSwap() will:
 *   - Check your token balance
 *   - Approve exactly the input amount to the AFI contract (skipped if already approved)
 *   - Simulate the swap via eth_call — throws before sending if it would revert
 *   - Send the swap transaction
 *   - Return confirmed amounts from the on-chain SwapExecuted event
 */
import { AfiClient, SimulationFailedError, InsufficientBalanceError } from "@afi-run/sdk"

const USDC = "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
const WETH = "0x4200000000000000000000000000000000000006"

const client = new AfiClient({
  rpcUrl: "https://rpc.ankr.com/base/YOUR_API_KEY",
  privateKey: "0xYOUR_PRIVATE_KEY",
})

async function main() {
  // Step 1: Get quote (read-only, no tx)
  console.log("Step 1: Fetching quote...")
  const quote = await client.getQuote({
    tokenIn: USDC,
    tokenOut: WETH,
    amountIn: 1000_000000n, // 1000 USDC
    slippage: 0.5,
  })

  const amountOutEth = Number(quote.amountOutWei) / 1e18
  const minOutEth = Number(quote.minOutWei) / 1e18

  // Step 2: Show pricing
  console.log("\nStep 2: Quote received:")
  console.log(`  Expected:  ~${amountOutEth.toFixed(6)} WETH`)
  console.log(`  Minimum:   ${minOutEth.toFixed(6)} WETH (${quote.slippage}% slippage)`)
  console.log(`  Fee:       ${quote.feeBps} bps`)

  // Step 3: User confirms — in a real app this would be a UI prompt
  const userConfirmed = true
  if (!userConfirmed) {
    console.log("\nSwap cancelled by user.")
    return
  }

  // Step 4: Execute with the existing quote (no second fetch needed)
  console.log("\nStep 3: Executing swap...")
  try {
    const result = await client.executeSwap(quote)

    const actualOutEth = Number(result.amountOut) / 1e18

    console.log("\nSwap confirmed!")
    console.log(`  Tx hash:    ${result.txHash}`)
    console.log(`  Block:      ${result.blockNumber}`)
    console.log(`  Actual out: ${actualOutEth.toFixed(6)} WETH  (from on-chain event)`)
    console.log(`  Gas used:   ${result.gasUsed}`)
  } catch (e) {
    if (e instanceof InsufficientBalanceError) {
      console.error(`Not enough USDC. Have: ${e.balance}, need: ${e.required}`)
    } else if (e instanceof SimulationFailedError) {
      // Swap would revert — no tx was sent, no gas wasted
      console.error(`Swap would revert: ${e.reason}`)
    } else {
      throw e
    }
  }
}

main().catch(console.error)
