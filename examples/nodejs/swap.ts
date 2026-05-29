import { AfiClient, formatUnits, InsufficientBalanceError, SimulationFailedError } from "@afi-run/sdk"

const USDC = "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
const WETH = "0x4200000000000000000000000000000000000006"

const client = new AfiClient({
  rpcUrl: "https://rpc.ankr.com/<YOUR_API_KEY>",
}).connect("0x<YOUR_PRIVATE_KEY>")

async function main() {
  // Swap 100 USDC → WETH with 0.5% slippage
  // amountIn is a human-readable string — "100" means 100 USDC
  try {
    const result = await client.swap({
      tokenIn: USDC,
      tokenOut: WETH,
      amountIn: "100",
      slippage: 0.5,
    })

    console.log("Swap successful!")
    console.log("Tx hash:   ", result.txHash)
    console.log("Block:     ", result.blockNumber)
    console.log("Amount in: ", formatUnits(result.amountIn, 6), "USDC")
    console.log("Amount out:", formatUnits(result.amountOut, 18), "WETH")
    console.log("Gas used:  ", result.gasUsed)
  } catch (e) {
    if (e instanceof InsufficientBalanceError) {
      console.error("Not enough balance:", e.message)
    } else if (e instanceof SimulationFailedError) {
      console.error("Swap would revert:", e.reason)
    } else {
      throw e
    }
  }
}

// Get a quote without executing — no private key needed for this
async function quoteOnly() {
  const readOnlyClient = new AfiClient({
    rpcUrl: "https://rpc.ankr.com/<YOUR_API_KEY>",
  })

  const quote = await readOnlyClient.getQuote({
    tokenIn: USDC,
    tokenOut: WETH,
    amountIn: "100",
    slippage: 0.5,
  })

  console.log("Quote:")
  console.log("  Expected out:", quote.amountOut, "WETH")
  console.log("  Min out:     ", quote.minOut, "WETH  (with slippage)")
  console.log("  Fee:         ", quote.feeBps, "bps")
  console.log("  Path:        ", quote.path.join(" → "))
}

main().catch(console.error)
