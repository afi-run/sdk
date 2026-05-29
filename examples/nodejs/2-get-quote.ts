/**
 * Example 2: Fetch a quote and inspect it before swapping
 *
 * client.quote() returns a QuoteBuilder — chain methods to configure,
 * then call .get() to fetch the quote. No private key needed.
 */
import { AfiClient, NETWORK, DEX } from "@afi-run/sdk"

const USDC = "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
const WETH = "0x4200000000000000000000000000000000000006"

const client = new AfiClient({
  rpcUrl: "https://rpc.ankr.com/base/YOUR_API_KEY",
})

async function main() {
  console.log("Fetching quote for 1000 USDC → WETH...\n")

  // Basic quote
  const quote = await client
    .quote(USDC, WETH, "1000")
    .slippage(0.5)
    .get()

  console.log("Basic quote:")
  console.log(`  You send:     ${quote.amountIn} USDC`)
  console.log(`  You receive:  ~${quote.amountOut} WETH`)
  console.log(`  Minimum out:  ${quote.minOut} WETH (${quote.slippage}% slippage)`)
  console.log(`  Protocol fee: ${(quote.feeBps / 100).toFixed(2)}%  (${quote.feeBps} bps)`)
  console.log(`  Route:        ${quote.path.join(" → ")}`)
  console.log(`  Hops:         ${quote.hops.length}`)
  for (const hop of quote.hops) {
    console.log(`    [${hop.type}/${hop.kind}] ${hop.amountIn} → ${hop.amountOut}`)
  }

  // Quote with optional parameters
  console.log("\nQuote with priceBase and specific DEXes:")
  const detailedQuote = await client
    .quote(USDC, WETH, "500")
    .slippage(0.5)
    .maxHops(3)
    .network(NETWORK.BASE)
    .priceBase("USDC")                    // adds tokenInBasePrice / tokenOutBasePrice
    .dexs(DEX.UNI_V3, DEX.AERODROME)     // restrict to these DEXes
    .get()

  console.log(`  You send:          ${detailedQuote.amountIn} USDC`)
  console.log(`  You receive:       ~${detailedQuote.amountOut} WETH`)
  if (detailedQuote.tokenInBasePrice) {
    console.log(`  USDC base price:   $${detailedQuote.tokenInBasePrice}`)
    console.log(`  WETH base price:   $${detailedQuote.tokenOutBasePrice}`)
  }

  // Pass Token objects directly from getTokens()
  console.log("\nQuote using Token objects from getTokens():")
  const tokens = await client.getTokens()
  const usdcToken = tokens.find((t) => t.symbol === "USDC")!
  const wethToken = tokens.find((t) => t.symbol === "WETH")!

  const tokenQuote = await client
    .quote(usdcToken, wethToken, "100")   // pass Token directly — no address extraction needed
    .slippage(1.0)
    .get()

  console.log(`  ${tokenQuote.amountIn} USDC → ~${tokenQuote.amountOut} WETH`)
  console.log("\nQuote ready. Pass to client.executeSwap(quote) to execute.")
}

main().catch(console.error)
