/**
 * Example 5: Approve only (for UIs that manage the flow step by step)
 *
 * Some UIs need to separate the approve step from the swap step —
 * for example, to show two separate wallet prompts to the user.
 *
 * approve() approves exactly the amount from the quote — no more.
 * If the allowance is already sufficient, no transaction is sent (returns null).
 */
import { AfiClient } from "@afi-run/sdk"

const USDC = "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
const WETH = "0x4200000000000000000000000000000000000006"

const client = new AfiClient({
  rpcUrl: "https://rpc.ankr.com/base/YOUR_API_KEY",
  privateKey: "0xYOUR_PRIVATE_KEY",
})

async function main() {
  // 1. Get quote first — the quote tells us exactly how much to approve
  const quote = await client.getQuote({
    tokenIn: USDC,
    tokenOut: WETH,
    amountIn: 200_000000n, // 200 USDC
    slippage: 0.5,
  })

  // 2. Approve step (wallet prompt #1)
  console.log(`Approving ${quote.amountInWei} wei USDC...`)
  const approveTx = await client.approve(USDC, quote.amountInWei)

  if (approveTx === null) {
    console.log("Allowance already sufficient — approval skipped.")
  } else {
    console.log(`Approval tx: ${approveTx}`)
  }

  // 3. Swap step (wallet prompt #2)
  // executeSwap() re-checks allowance internally before sending,
  // so it's safe to call even if some time passed since approve.
  console.log("Executing swap...")
  const result = await client.executeSwap(quote)

  console.log(`Swap confirmed: ${result.txHash}`)
}

main().catch(console.error)
