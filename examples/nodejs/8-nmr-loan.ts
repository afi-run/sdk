/**
 * Example 8: NMR user-funded arbitrage via NMR.loan(...)
 *
 * Unlike requestOperation (flash-loan funded), `loan(user, asset, amount, minOut, params)`
 * uses the *user's* tokens. The user must approve NMR for `amount` of `asset` first —
 * the operator can NOT bypass this. NMR pulls the funds, runs the cycle, returns
 * `out >= minOut` to the user, and keeps the operator share.
 *
 * Required approval (run from the user's wallet before the operator submits):
 *
 *     IERC20(asset).approve(NMR_ADDRESS, amount)
 *
 * The example shows that approval explicitly when invoked with USER_PRIVATE_KEY.
 *
 * Prerequisites:
 *  - RPC_URL                — Base RPC endpoint
 *  - USER_PRIVATE_KEY       — funds the trade (must hold `amount` of `asset`)
 *  - OPERATOR_PRIVATE_KEY   — operator submitting the loan call (one of the 7)
 *
 * Expected output: approval tx, loan tx, parsed FlashLoanExecuted event, net out.
 *
 * NOTE: high-level `client.nmrLoanArbitrage()` not yet exposed.
 * (TODO: client.nmrLoanArbitrage when SDK lands it.)
 */
import {
  AfiClient,
  NMR_ADDRESSES,
  NMR_ABI,
  ERC20_ABI,
  encodeSteps,
  parseFlashLoanExecuted,
  formatUnits,
  type Address,
  type Hex,
  type Step,
} from "@afi-run/sdk"
import { createWalletClient, http } from "viem"
import { privateKeyToAccount } from "viem/accounts"
import { base } from "viem/chains"

const RPC_URL      = process.env.RPC_URL              ?? "https://mainnet.base.org"
const USER_KEY     = process.env.USER_PRIVATE_KEY     as Hex
const OPERATOR_KEY = process.env.OPERATOR_PRIVATE_KEY as Hex

const USDC: Address = "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
const NMR            = NMR_ADDRESSES[8453]

// Hard-coded for this example. Replace with a real route from findArbitrage().
const STEPS: Step[] = [
  { id: 3, data: "0x" as Hex }, // UniV3 placeholder
  { id: 5, data: "0x" as Hex }, // Aerodrome placeholder
]

async function main() {
  if (!USER_KEY)     throw new Error("Set USER_PRIVATE_KEY env var")
  if (!OPERATOR_KEY) throw new Error("Set OPERATOR_PRIVATE_KEY env var")

  const client   = new AfiClient({ rpcUrl: RPC_URL })
  const userAcc  = privateKeyToAccount(USER_KEY)
  const opAcc    = privateKeyToAccount(OPERATOR_KEY)

  const userWallet = createWalletClient({ account: userAcc, chain: base, transport: http(RPC_URL) })
  const opWallet   = createWalletClient({ account: opAcc,   chain: base, transport: http(RPC_URL) })

  console.log(`User:     ${userAcc.address}`)
  console.log(`Operator: ${opAcc.address}`)
  console.log(`NMR:      ${NMR}\n`)

  const amountWei = 1_000_000_000n // 1000 USDC (6 decimals)
  const minOutWei = 1_005_000_000n // require at least 0.5% gain
  const params    = encodeSteps(STEPS)

  // Step 1: user.approve(NMR, amount) — MANDATORY.
  console.log("Step 1: user approves NMR to pull USDC...")
  const approveHash = await userWallet.writeContract({
    address:      USDC,
    abi:          ERC20_ABI,
    functionName: "approve",
    args:         [NMR, amountWei],
  })
  await client.waitForTx(approveHash)
  console.log(`  approved: ${approveHash}\n`)

  // Step 2: operator submits the loan call. NMR transfers `amount` USDC from the
  // user, runs the cycle, and returns `out >= minOut` to the user.
  console.log("Step 2: operator submits NMR.loan(...)")
  // TODO: client.nmrLoanArbitrage when SDK lands it.
  const loanHash = await opWallet.writeContract({
    address:      NMR,
    abi:          NMR_ABI,
    functionName: "loan",
    args:         [userAcc.address, USDC, amountWei, minOutWei, params],
  })
  console.log(`  loan tx: ${loanHash}`)

  const receipt = await client.waitForTx(loanHash)
  console.log(`  confirmed block ${receipt.blockNumber}, gas ${receipt.gasUsed}\n`)

  const pubReceipt = await (client as unknown as { pub: { getTransactionReceipt: (args: { hash: Hex }) => Promise<{ logs: never[] }> } }).pub
    .getTransactionReceipt({ hash: loanHash })
  for (const ev of parseFlashLoanExecuted(pubReceipt.logs)) {
    console.log(`FlashLoanExecuted:`)
    console.log(`  amount:  ${formatUnits(ev.amount, 6)} USDC`)
    console.log(`  premium: ${formatUnits(ev.premium, 6)} USDC`)
    console.log(`  profit:  ${formatUnits(ev.profit, 6)} USDC`)
  }
}

main().catch((e) => { console.error(e); process.exit(1) })
