/**
 * Example 9: Owner-only admin governance flow
 *
 * Reads the Afi router state, then exercises three owner-restricted operations:
 *   1. adminPause()           — halt all swap/swapFor traffic
 *   2. adminSetFeeBps(25)     — change the default protocol fee
 *   3. adminAddRule(rule)     — append a custom routing rule
 *
 * WARNING: only the contract owner can run this. The OWNER_PRIVATE_KEY env var
 * must hold the key for the address returned by `Afi.owner()`. If you wire in
 * the wrong key, every tx reverts with OwnableUnauthorizedAccount.
 *
 * Prerequisites:
 *  - RPC_URL              — Base RPC endpoint
 *  - OWNER_PRIVATE_KEY    — Afi contract owner (Base)
 *  - RULE_ADDR (optional) — address of a deployed IRule to add
 *
 * Expected output: pre-state, three tx hashes, post-state.
 *
 * NOTE: convenience wrappers like `client.adminPause()` are not exposed yet.
 * This uses AFI_ABI + viem writeContract as the fallback.
 * (TODO: client.adminPause / adminSetFeeBps / adminAddRule when SDK lands them.)
 */
import {
  AfiClient,
  AFI_ADDRESSES,
  AFI_ABI,
  type Address,
  type Hex,
} from "@afi-run/sdk"
import { createWalletClient, http } from "viem"
import { privateKeyToAccount } from "viem/accounts"
import { base } from "viem/chains"

const RPC_URL    = process.env.RPC_URL              ?? "https://mainnet.base.org"
const OWNER_KEY  = process.env.OWNER_PRIVATE_KEY    as Hex
const RULE_ADDR  = (process.env.RULE_ADDR           ?? "0x0000000000000000000000000000000000000000") as Address
const SAMPLE_USER: Address = "0x1111111111111111111111111111111111111111"

const AFI = AFI_ADDRESSES[8453]

async function main() {
  if (!OWNER_KEY) throw new Error("Set OWNER_PRIVATE_KEY env var")

  const client = new AfiClient({ rpcUrl: RPC_URL }).connect(OWNER_KEY)
  const owner  = privateKeyToAccount(OWNER_KEY)
  const wallet = createWalletClient({ account: owner, chain: base, transport: http(RPC_URL) })

  console.log(`Owner: ${owner.address}`)
  console.log(`AFI:   ${AFI}\n`)

  // ─── Pre-state ──────────────────────────────────────────────────────────────
  const pub = (client as unknown as { pub: {
    readContract: (args: { address: Address; abi: typeof AFI_ABI; functionName: string; args?: unknown[] }) => Promise<unknown>
  } }).pub

  const paused0 = await pub.readContract({ address: AFI, abi: AFI_ABI, functionName: "paused" })
  const fee0    = await pub.readContract({ address: AFI, abi: AFI_ABI, functionName: "feeBps" })
  const userFee = await pub.readContract({ address: AFI, abi: AFI_ABI, functionName: "feeBpsOf", args: [SAMPLE_USER] })

  console.log(`Pre-state:`)
  console.log(`  paused:           ${paused0}`)
  console.log(`  default feeBps:   ${fee0}`)
  console.log(`  user feeBps:      ${userFee}\n`)

  // ─── 1. Pause ───────────────────────────────────────────────────────────────
  // TODO: client.adminPause when SDK lands it.
  const pauseHash = await wallet.writeContract({
    address: AFI, abi: AFI_ABI, functionName: "pause", args: [],
  })
  console.log(`adminPause()                 → ${pauseHash}`)
  await client.waitForTx(pauseHash)

  // ─── 2. setFeeBps(25) ───────────────────────────────────────────────────────
  // TODO: client.adminSetFeeBps when SDK lands it.
  const feeHash = await wallet.writeContract({
    address: AFI, abi: AFI_ABI, functionName: "setFeeBps", args: [25],
  })
  console.log(`adminSetFeeBps(25)           → ${feeHash}`)
  await client.waitForTx(feeHash)

  // ─── 3. addRule(addr) ───────────────────────────────────────────────────────
  // Only attempted when RULE_ADDR is set to a non-zero address.
  if (RULE_ADDR !== "0x0000000000000000000000000000000000000000") {
    // TODO: client.adminAddRule when SDK lands it.
    const ruleHash = await wallet.writeContract({
      address: AFI, abi: AFI_ABI, functionName: "addRule", args: [RULE_ADDR],
    })
    console.log(`adminAddRule(${RULE_ADDR}) → ${ruleHash}`)
    await client.waitForTx(ruleHash)
  } else {
    console.log("adminAddRule() skipped — set RULE_ADDR to attempt it")
  }

  // ─── Post-state ─────────────────────────────────────────────────────────────
  const paused1 = await pub.readContract({ address: AFI, abi: AFI_ABI, functionName: "paused" })
  const fee1    = await pub.readContract({ address: AFI, abi: AFI_ABI, functionName: "feeBps" })

  console.log(`\nPost-state:`)
  console.log(`  paused:         ${paused1}`)
  console.log(`  default feeBps: ${fee1}`)
}

main().catch((e) => { console.error(e); process.exit(1) })
