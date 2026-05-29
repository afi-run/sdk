/**
 * Example 3: Quote → review → execute (recommended flow)
 *
 * 1. Fetch a quote with .get() — no tx sent
 * 2. Show pricing to the user
 * 3. User confirms → call executeSwap(quote)
 *
 * executeSwap() handles: balance check → approve → simulate → swap
 */
import { AfiClient, SimulationFailedError, InsufficientBalanceError, formatUnits } from "@afi-run/sdk"

const USDC = "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
const WETH = "0x4200000000000000000000000000000000000006"

const client = new AfiClient({
  rpcUrl: "https://rpc.ankr.com/base/YOUR_API_KEY",
})

async function main() {
  // Step 1: Fetch quote — no private key needed
  console.log("Step 1: Fetching quote...")
  const quote = await client
    .quote(USDC, WETH, "1000")
    .slippage(0.5)
    .get()

  // Step 2: Show pricing to user
  console.log("\nStep 2: Quote received:")
  console.log(`  Expected:  ~${quote.amountOut} WETH  ($${quote.tokenOutPrice})`)
  console.log(`  Minimum:   ${quote.minOut} WETH (${quote.slippage}% slippage)`)
  console.log(`  Fee:       ${quote.feeBps} bps`)

  // Step 3: Connect signer and execute
  const userConfirmed = true
  if (!userConfirmed) { console.log("\nCancelled."); return }

  client.connect("0xYOUR_PRIVATE_KEY")
  console.log("\nStep 3: Executing swap...")

  try {
    const result = await client.executeSwap(quote)
    console.log("\nSwap confirmed!")
    console.log(`  Tx hash:    ${result.txHash}`)
    console.log(`  Block:      ${result.blockNumber}`)
    console.log(`  Actual out: ${formatUnits(result.amountOut, 18)} WETH`)
    console.log(`  Gas used:   ${result.gasUsed}`)
  } catch (e) {
    if (e instanceof InsufficientBalanceError) {
      console.error(`Not enough USDC: have ${formatUnits(e.balance, 6)}, need ${formatUnits(e.required, 6)}`)
    } else if (e instanceof SimulationFailedError) {
      console.error(`Swap would revert: ${e.reason}`)
    } else {
      throw e
    }
  }
}

main().catch(console.error)
