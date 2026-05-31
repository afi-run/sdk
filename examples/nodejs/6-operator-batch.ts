/**
 * Example 6: Operator batchSwapFor — execute swaps on behalf of pre-approved users
 *
 * The Afi router exposes `swapFor(user, ...)` and `batchSwapFor(tuple[])` which
 * can only be called by an account registered as an operator on the contract.
 * Each `user` in the batch must have approved the Afi router for `amountIn` of
 * `tokenIn` ahead of time.
 *
 * Prerequisites:
 *  - RPC_URL                — Base RPC endpoint
 *  - OPERATOR_PRIVATE_KEY   — one of the 7 registered operators on Afi (Base)
 *  - USER1_ADDR, USER2_ADDR, USER3_ADDR — pre-approved users
 *
 * Expected output: one tx hash, plus a per-user SwapExecuted row parsed from
 * the receipt logs.
 *
 * NOTE: high-level `client.batchSwapFor()` is not yet exposed by the SDK — this
 * uses the AFI_ABI + viem's writeContract as the fallback. Swap to the high-level
 * helper when it lands. (TODO: client.batchSwapFor when SDK lands it.)
 */
import {
  AfiClient,
  AFI_ADDRESSES,
  AFI_ABI,
  parseSwapExecuted,
  formatUnits,
  encodeSteps,
  type Address,
  type Hex,
} from "@afi-run/sdk"
import { createWalletClient, http } from "viem"
import { privateKeyToAccount } from "viem/accounts"
import { base } from "viem/chains"

const RPC_URL          = process.env.RPC_URL          ?? "https://mainnet.base.org"
const OPERATOR_KEY     = process.env.OPERATOR_PRIVATE_KEY as Hex
const USER1            = (process.env.USER1_ADDR ?? "0x0000000000000000000000000000000000000001") as Address
const USER2            = (process.env.USER2_ADDR ?? "0x0000000000000000000000000000000000000002") as Address
const USER3            = (process.env.USER3_ADDR ?? "0x0000000000000000000000000000000000000003") as Address

const USDC: Address = "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
const WETH: Address = "0x4200000000000000000000000000000000000006"
const AFI            = AFI_ADDRESSES[8453]

interface SwapRequest {
  user:     Address
  tokenIn:  Address
  amountIn: bigint
  tokenOut: Address
  minOut:   bigint
  params:   Hex
}

async function main() {
  if (!OPERATOR_KEY) throw new Error("Set OPERATOR_PRIVATE_KEY env var")

  const client = new AfiClient({ rpcUrl: RPC_URL }).connect(OPERATOR_KEY)
  const operator = privateKeyToAccount(OPERATOR_KEY)
  const wallet = createWalletClient({ account: operator, chain: base, transport: http(RPC_URL) })

  console.log(`Operator: ${operator.address}`)
  console.log(`AFI:      ${AFI}\n`)

  // Build one SwapRequest per user using the SDK quoter.
  const users  = [USER1, USER2, USER3]
  const amount = "100" // 100 USDC each

  const reqs: SwapRequest[] = []
  for (const user of users) {
    const quote = await client.quote(USDC, WETH, amount).slippage(0.5).get()
    console.log(`Quote for ${user}: ${quote.amountIn} USDC → ~${quote.amountOut} WETH`)
    reqs.push({
      user,
      tokenIn:  USDC,
      amountIn: quote.amountInWei,
      tokenOut: WETH,
      minOut:   quote.minOutWei,
      params:   encodeSteps(quote.steps as never) as Hex,
    })
  }

  // Submit batchSwapFor (operator-only on chain).
  // TODO: high-level client.batchSwapFor when SDK lands it.
  const txHash = await wallet.writeContract({
    address:      AFI,
    abi:          AFI_ABI,
    functionName: "batchSwapFor",
    args:         [reqs],
  })
  console.log(`\nbatchSwapFor submitted: ${txHash}`)

  const receipt = await client.waitForTx(txHash)
  console.log(`Confirmed in block ${receipt.blockNumber}, gas used ${receipt.gasUsed}`)

  // Parse one SwapExecuted per user from the receipt logs.
  const pubReceipt = await (client as unknown as { pub: { getTransactionReceipt: (args: { hash: Hex }) => Promise<{ logs: never[] }> } }).pub
    .getTransactionReceipt({ hash: txHash })
  const events = parseSwapExecuted(pubReceipt.logs)
  console.log(`\n${events.length} SwapExecuted events:`)
  for (const ev of events) {
    console.log(`  ${ev.from}  ${formatUnits(ev.amountIn, 6)} USDC → ${formatUnits(ev.amountOut, 18)} WETH`)
  }
}

main().catch((e) => { console.error(e); process.exit(1) })
