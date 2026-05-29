/**
 * Example 5: Staged flow — full control over each step
 *
 * 1. Fetch quote with .get()
 * 2. Connect signer
 * 3. Approve  — get txHash immediately, wait separately
 * 4. Simulate — dry-run before spending gas
 * 5. Submit   — send the swap tx, get txHash immediately
 * 6. Wait     — block until confirmed, get SwapResult
 *
 * Use this in apps that need step-by-step wallet prompts and progress UI.
 */
import { AfiClient, NETWORK, formatUnits, InsufficientBalanceError } from "@afi-run/sdk"

const USDC = "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
const WETH = "0x4200000000000000000000000000000000000006"

async function main() {
  // Step 1: Read-only client — fetch quote without private key
  const client = new AfiClient({ rpcUrl: "https://rpc.ankr.com/base/YOUR_API_KEY" })

  const quote = await client
    .quote(USDC, WETH, "200")
    .slippage(0.5)
    .network(NETWORK.BASE)
    .get()

  console.log(`Quote: ${quote.amountIn} USDC → ~${quote.amountOut} WETH`)
  console.log(`Minimum: ${quote.minOut} WETH  (${quote.slippage}% slippage)`)

  // Step 2: Connect signer when user is ready to transact
  client.connect("0xYOUR_PRIVATE_KEY")

  // Step 3: Approve — txHash available immediately, wait separately
  const approval = await client.approve(quote.tokenIn, quote.amountInWei)

  if (approval === null) {
    console.log("Allowance already sufficient — approval skipped.")
  } else {
    console.log(`Approval submitted: ${approval.txHash}`)
    const approvalReceipt = await approval.wait()
    console.log(`Approval confirmed in block ${approvalReceipt.blockNumber}`)
  }

  // Step 4: Simulate — check before spending gas
  const ok = await client.simulate(quote, (reason) => {
    console.error(`Simulation failed: ${reason}`)
  })
  if (!ok) { console.log("Swap would revert — cancelled."); return }

  // Step 5: Submit — send the swap tx, get txHash immediately
  const pending = await client.submitSwap(quote)
  console.log(`Swap submitted: ${pending.txHash}`)

  // Step 6: Wait for on-chain confirmation
  const result = await pending.wait()

  console.log("\nSwap confirmed!")
  console.log(`  Tx hash:    ${result.txHash}`)
  console.log(`  Block:      ${result.blockNumber}`)
  console.log(`  Amount out: ${formatUnits(result.amountOut, 18)} WETH`)
  console.log(`  Gas used:   ${result.gasUsed}`)
}

main().catch(console.error)
