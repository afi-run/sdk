/**
 * Example 12: AfiReferralRouter — referral fee + delegation
 *
 * The AfiReferralRouter wraps Afi.swap and can charge a referral fee of up to
 * 0.10% on the OUTPUT token, credited to a referrer and claimed later. It also
 * supports delegated swaps (B swaps A's funds; output goes to A).
 *
 * This example:
 *   1. Resolves the router address for the chain.
 *   2. Prints swapWithReferral calldata (needs Afi route `params` + an ERC20
 *      approval to the router — see examples 2/3 for how to build `params`).
 *   3. Sends a real setDelegateAllowance tx (touches only the caller's own
 *      mapping — cheap and safe), then revokes it.
 *   4. Prints owner-only setMaxReferralBps calldata (NOT sent).
 *
 * Prerequisites:
 *   - RPC_URL      — Base RPC endpoint
 *   - PRIVATE_KEY  — any funded key (the delegation txs spend only gas)
 *
 * Run: tsx examples/nodejs/12-referral-swap.ts
 */
import {
  referralRouterAddress,
  encodeSwapWithReferral,
  encodeSetDelegateAllowance,
  encodeRevokeDelegate,
  encodeReferralSetMaxReferralBps,
  REFERRAL_HARD_CAP_BPS,
  type Address,
  type Hex,
} from "@afi-run/sdk"
import { createWalletClient, http } from "viem"
import { privateKeyToAccount } from "viem/accounts"
import { base } from "viem/chains"

const RPC_URL = process.env.RPC_URL ?? "https://mainnet.base.org"
const PK = process.env.PRIVATE_KEY as Hex

const USDC: Address     = "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
const WETH: Address     = "0x4200000000000000000000000000000000000006"
const REFERRER: Address = "0x1111111111111111111111111111111111111111"
const DELEGATE: Address = "0x2222222222222222222222222222222222222222"

async function main() {
  if (!PK) throw new Error("Set PRIVATE_KEY env var")

  const account = privateKeyToAccount(PK)
  const wallet = createWalletClient({ account, chain: base, transport: http(RPC_URL) })

  // 1. Resolve the router for this chain (Base = 8453).
  const router = referralRouterAddress(8453)
  console.log(`Router: ${router}`)
  console.log(`Caller: ${account.address}\n`)

  // 2. swapWithReferral calldata (not sent — needs real Afi `params` + an ERC20
  //    approval of `tokenIn` to the router).
  const swapData = encodeSwapWithReferral({
    tokenIn: USDC,
    amountIn: 1_000_000n, // 1 USDC (6 decimals)
    tokenOut: WETH,
    minOut: 0n,           // set this to your slippage-protected minimum (net of fee)
    params: "0x",         // Afi route params — build with the step builders
    referrer: REFERRER,   // zero address disables the fee
    referralBps: REFERRAL_HARD_CAP_BPS, // <= 10
  })
  console.log("swapWithReferral calldata (send to router with a real `params`):")
  console.log(`  ${swapData}\n`)

  // 3. Authorize a delegate to spend up to 5 USDC until a far-future deadline,
  //    then revoke. These only touch the caller's own allowance mapping.
  const deadline = 1893456000 // 2030-01-01 (unix seconds, uint48)
  const setData = encodeSetDelegateAllowance(USDC, DELEGATE, 5_000_000n, deadline)
  const h1 = await wallet.sendTransaction({ to: router, data: setData })
  console.log(`setDelegateAllowance → ${h1}`)

  const revokeData = encodeRevokeDelegate(USDC, DELEGATE)
  const h2 = await wallet.sendTransaction({ to: router, data: revokeData })
  console.log(`revokeDelegate       → ${h2}\n`)

  // 4. Owner-only: lower the effective referral cap to 5 bps (calldata only).
  const capData = encodeReferralSetMaxReferralBps(5)
  console.log("setMaxReferralBps(5) calldata (owner-only, NOT sent):")
  console.log(`  ${capData}`)
}

main().catch((err) => {
  console.error(err)
  process.exit(1)
})
