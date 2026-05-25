import { AfiClient, InsufficientBalanceError, SimulationFailedError } from "@afi-run/sdk"

const USDC = "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
const WETH = "0x4200000000000000000000000000000000000006"

const client = new AfiClient({
  rpcUrl: "https://rpc.ankr.com/base/<YOUR_API_KEY>",
  privateKey: "0x<YOUR_PRIVATE_KEY>",
})

async function main() {
  // Swap 100 USDC → WETH with 0.5% slippage
  try {
    const result = await client.swap({
      tokenIn: USDC,
      tokenOut: WETH,
      amountIn: 100_000000n, // 100 USDC (6 decimals)
      slippage: 0.5,
    })

    console.log("Swap successful!")
    console.log("Tx hash:   ", result.txHash)
    console.log("Block:     ", result.blockNumber)
    console.log("Amount in: ", result.amountIn)
    console.log("Amount out:", result.amountOut)
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

// Get a quote without executing
async function quoteOnly() {
  const quote = await client.getQuote({
    tokenIn: USDC,
    tokenOut: WETH,
    amountIn: 100_000000n,
    slippage: 0.5,
  })

  console.log("Quote:")
  console.log("  Expected out:", quote.amountOutWei, "wei")
  console.log("  Min out:     ", quote.minOutWei, "wei  (with slippage)")
  console.log("  Fee:         ", quote.feeBps, "bps")
  console.log("  Path:        ", quote.path.join(" → "))
}

main().catch(console.error)
