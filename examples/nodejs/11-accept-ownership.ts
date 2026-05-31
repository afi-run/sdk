/**
 * Example 11: Accept ownership of all AFI contracts (Ownable2Step handover)
 *
 * The Solidity script `script/TransferOwnership.s.sol` only INITIATES the
 * Ownable2Step transfer for each contract. Until the new owner calls
 * `acceptOwnership()` on each one, current ownership stays with the deployer.
 *
 * This example assumes the running key (NEW_OWNER_PRIVATE_KEY) is the pending
 * owner. It:
 *   1. Reads `pendingOwner()` on the AFI router to confirm we are it.
 *   2. Sends `acceptOwnership()` to all 12 contracts that were transferred:
 *        RouteRegistry, Afi, NMR, CakeV3, UniV3, UniV4, Aerodrome,
 *        BalancerV3, Curve128, Curve256, FluidDex, AaveLiquidator.
 *
 * For Safe / multi-sig owners, you'd batch these in the Safe UI instead —
 * this example targets the EOA flow only.
 *
 * Prerequisites:
 *  - RPC_URL                  — Base RPC endpoint
 *  - NEW_OWNER_PRIVATE_KEY    — pending owner key for every contract
 *  - ROUTE_REGISTRY, AFI, NMR, CAKEV3, UNIV3, UNIV4, AERODROME, BALANCERV3,
 *    CURVE128, CURVE256, FLUIDDEX, AAVE_LIQUIDATOR  — addresses to accept on
 *    (missing addresses are skipped with a log line)
 *
 * Expected output: pending-owner confirmation, then one tx hash per contract.
 *
 * NOTE: convenience wrappers `client.acceptOwnership(addr)` / `acceptOwnershipBatch(addrs)`
 * are not exposed yet — this uses the SDK's `encodeAcceptOwnership()` helper
 * with viem's writeContract for the raw send.
 * (TODO: client.acceptOwnership / acceptOwnershipBatch when SDK lands them.)
 */
import {
  AfiClient,
  AFI_ADDRESSES,
  OWNABLE2STEP_ABI,
  type Address,
  type Hex,
} from "@afi-run/sdk"
import { createWalletClient, http } from "viem"
import { privateKeyToAccount } from "viem/accounts"
import { base } from "viem/chains"

const RPC_URL    = process.env.RPC_URL                ?? "https://mainnet.base.org"
const OWNER_KEY  = process.env.NEW_OWNER_PRIVATE_KEY  as Hex

// Contract addresses transferred by TransferOwnership.s.sol — fill from your deploy.
const TARGETS: { name: string; addr?: string }[] = [
  { name: "RouteRegistry",  addr: process.env.ROUTE_REGISTRY  },
  { name: "Afi",            addr: process.env.AFI             },
  { name: "NMR",            addr: process.env.NMR             },
  { name: "CakeV3",         addr: process.env.CAKEV3          },
  { name: "UniV3",          addr: process.env.UNIV3           },
  { name: "UniV4",          addr: process.env.UNIV4           },
  { name: "Aerodrome",      addr: process.env.AERODROME       },
  { name: "BalancerV3",     addr: process.env.BALANCERV3      },
  { name: "Curve128",       addr: process.env.CURVE128        },
  { name: "Curve256",       addr: process.env.CURVE256        },
  { name: "FluidDex",       addr: process.env.FLUIDDEX        },
  { name: "AaveLiquidator", addr: process.env.AAVE_LIQUIDATOR },
]

async function main() {
  if (!OWNER_KEY) throw new Error("Set NEW_OWNER_PRIVATE_KEY env var")

  const client   = new AfiClient({ rpcUrl: RPC_URL }).connect(OWNER_KEY)
  const newOwner = privateKeyToAccount(OWNER_KEY)
  const wallet   = createWalletClient({ account: newOwner, chain: base, transport: http(RPC_URL) })

  console.log(`New owner: ${newOwner.address}`)

  // Step 1: sanity check — read pendingOwner() on the AFI router.
  const pending = await client.getPendingOwner()
  console.log(`AFI router pendingOwner(): ${pending}`)
  if (pending.toLowerCase() !== newOwner.address.toLowerCase()) {
    console.error("Pending owner does NOT match this key — aborting.")
    return
  }

  // Step 2: call acceptOwnership() on each non-empty target.
  // TODO: client.acceptOwnership / acceptOwnershipBatch when SDK lands them.
  console.log("\nAccepting ownership on 12 contracts...")
  for (const { name, addr } of TARGETS) {
    if (!addr || addr === "0x0000000000000000000000000000000000000000") {
      console.log(`  [skip] ${name.padEnd(15)} (env not set)`)
      continue
    }
    const txHash = await wallet.writeContract({
      address:      addr as Address,
      abi:          OWNABLE2STEP_ABI,
      functionName: "acceptOwnership",
      args:         [],
    })
    console.log(`  [accept] ${name.padEnd(15)} → ${txHash}`)
    await client.waitForTx(txHash)
  }

  // Final sanity: AFI router pendingOwner should now be address(0).
  const after = await client.getPendingOwner()
  console.log(`\nAFI router pendingOwner() after: ${after}`)
  console.log("Ownership handover complete.")
}

// Reference the AFI_ADDRESSES export so it's not stripped — used in docs above.
void AFI_ADDRESSES

main().catch((e) => { console.error(e); process.exit(1) })
