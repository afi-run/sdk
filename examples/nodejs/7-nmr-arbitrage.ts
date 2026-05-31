/**
 * Example 7: NMR flash-loan arbitrage (operator-only)
 *
 * Flow:
 *  1. client.findArbitrage queries /arbitrage for candidate cycle routes
 *     (tokenIn === tokenOut === asset).
 *  2. Pick the most profitable route (routeProfit) and build the Afi.swap params
 *     with quoteFromRoute (wraps the route's encoded hop).
 *  3. client.executeNMRArbitrage calls NMR.requestOperation(asset, amount, params)
 *     — Aave flash-loan + cycle, signed by a registered operator.
 *  4. Parse the FlashLoanExecuted event and log realized profit.
 *
 * Prerequisites:
 *  - RPC_URL                — Base RPC endpoint
 *  - OPERATOR_PRIVATE_KEY   — one of the registered operators on NMR (Base)
 *  - AFI_API_URL            — afi-rpc base URL (default: https://rpc.afi.run)
 */
import {
  AfiClient,
  NMR_ADDRESSES,
  quoteFromRoute,
  routeProfit,
  parseFlashLoanExecuted,
  parseFlashLoanFailed,
  parseFlashLoanFailedWithData,
  formatUnits,
  type Address,
  type Hex,
} from "@afi-run/sdk"
import { privateKeyToAccount } from "viem/accounts"

const RPC_URL = process.env.RPC_URL ?? "https://mainnet.base.org"
const OPERATOR_KEY = process.env.OPERATOR_PRIVATE_KEY as Hex
const API_URL = process.env.AFI_API_URL ?? "https://rpc.afi.run"

const USDC: Address = "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
const NMR = NMR_ADDRESSES[8453]

async function main() {
  if (!OPERATOR_KEY) throw new Error("Set OPERATOR_PRIVATE_KEY env var")

  const client = new AfiClient({ rpcUrl: RPC_URL }).setApiUrl(API_URL).connect(OPERATOR_KEY)
  console.log(`Operator: ${privateKeyToAccount(OPERATOR_KEY).address}`)
  console.log(`NMR:      ${NMR}\n`)

  // 1. Discover candidate cycle routes (tokenIn === tokenOut === USDC).
  const routes = await client.findArbitrage({
    network: "base",
    tokenIn: USDC,
    tokenOut: USDC,
    amountIn: "1000",
  })

  // 2. Pick the most profitable route.
  let best = null
  for (const r of routes) {
    const p = routeProfit(r)
    if (p === null || p <= 0n) continue
    if (best === null || p > routeProfit(best)!) best = r
  }
  if (best === null) {
    console.log("No profitable arbitrage opportunity right now.")
    return
  }
  console.log(`Best route: dex=${best.kind} id=${best.routeId}, est. profit=${routeProfit(best)} (base units)`)

  const quote = quoteFromRoute(best)

  // 3. Broadcast NMR.requestOperation via the high-level wrapper.
  const receipt = await client.executeNMRArbitrage({
    asset: quote.tokenIn,
    amount: quote.amountInWei,
    params: quote.steps,
  })
  console.log(`\nrequestOperation: ${receipt.transactionHash}`)
  console.log(`Confirmed in block ${receipt.blockNumber}, gas used ${receipt.gasUsed}`)

  // 4. Parse the result events.
  const failures = parseFlashLoanFailed(receipt.logs)
  if (failures.length > 0) {
    console.log(`FlashLoanFailed: ${failures[0].reason}`)
    return
  }
  const lowLevel = parseFlashLoanFailedWithData(receipt.logs)
  if (lowLevel.length > 0) {
    console.log(`FlashLoanFailedWithData: ${lowLevel[0].data}`)
    return
  }
  for (const ev of parseFlashLoanExecuted(receipt.logs)) {
    console.log(`\nFlashLoanExecuted:`)
    console.log(`  asset:    ${ev.asset}`)
    console.log(`  amount:   ${formatUnits(ev.amount, 6)} USDC`)
    console.log(`  premium:  ${formatUnits(ev.premium, 6)} USDC`)
    console.log(`  profit:   ${formatUnits(ev.profit, 6)} USDC`)
  }
}

main().catch((e) => {
  console.error(e)
  process.exit(1)
})
