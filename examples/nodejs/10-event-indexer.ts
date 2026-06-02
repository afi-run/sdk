/**
 * Example 10: On-chain event indexer
 *
 * Reads the last N blocks from Base and tabulates the protocol's main events:
 *   - SwapExecuted        (Afi)
 *   - FeeCollected        (Afi)
 *
 * Uses viem's publicClient.getLogs() per contract address, then runs each log
 * batch through the SDK's typed parsers.
 *
 * Prerequisites:
 *  - RPC_URL                  — Base RPC endpoint
 *  - BLOCK_RANGE (optional)   — how many blocks back from `latest` to scan (default: 5000)
 *
 * Expected output: aggregate counts and a tail of recent rows per event type.
 */
import {
  AfiClient,
  AFI_ADDRESSES,
  parseSwapExecuted,
  parseFeeCollected,
  type Address,
  type Hex,
} from "@afi-run/sdk"

const RPC_URL     = process.env.RPC_URL ?? "https://mainnet.base.org"
const BLOCK_RANGE = BigInt(process.env.BLOCK_RANGE ?? "5000")

const AFI = AFI_ADDRESSES[8453]

async function main() {
  const client = new AfiClient({ rpcUrl: RPC_URL })
  const pub    = (client as unknown as { pub: {
    getBlockNumber: () => Promise<bigint>
    getLogs: (args: { address: Address; fromBlock: bigint; toBlock: bigint }) => Promise<unknown[]>
  } }).pub

  const tip   = await pub.getBlockNumber()
  const from  = tip - BLOCK_RANGE + 1n
  console.log(`Scanning blocks ${from} → ${tip} (${BLOCK_RANGE} blocks)\n`)

  const afiLogs = await pub.getLogs({ address: AFI, fromBlock: from, toBlock: tip })

  const swaps        = parseSwapExecuted(afiLogs as never)
  const feeCollects  = parseFeeCollected(afiLogs as never)

  console.log(`Event counts (${BLOCK_RANGE} blocks):`)
  console.log(`  SwapExecuted        ${swaps.length}`)
  console.log(`  FeeCollected        ${feeCollects.length}\n`)

  // Show the last 3 of each (head of the latest-first slice).
  const tail = <T>(xs: T[], n: number) => xs.slice(Math.max(0, xs.length - n))

  if (swaps.length > 0) {
    console.log("Recent SwapExecuted:")
    for (const s of tail(swaps, 3)) {
      console.log(`  ${s.from} ${s.assetIn} → ${s.assetOut}  amount=${s.amountIn}/${s.amountOut}`)
    }
  }
}

main().catch((e: unknown) => { console.error(e); process.exit(1) })

// Suppress unused-import warning for Hex (kept for type-import parity with siblings).
void (null as unknown as Hex)
