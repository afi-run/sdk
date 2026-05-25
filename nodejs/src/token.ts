import type { Account, PublicClient, Transport, WalletClient } from "viem"
import type { base } from "viem/chains"
import { AFI_ADDRESS, ERC20_ABI } from "./constants.js"
import { ApprovalError, InsufficientBalanceError } from "./errors.js"
import type { Address } from "./types.js"

type BasePublicClient = PublicClient<Transport, typeof base>
type BaseWalletClient = WalletClient<Transport, typeof base, Account>

export async function getDecimals(token: Address, client: BasePublicClient): Promise<number> {
  return client.readContract({ address: token, abi: ERC20_ABI, functionName: "decimals" })
}

export async function getBalance(
  token: Address,
  owner: Address,
  client: BasePublicClient,
): Promise<bigint> {
  return client.readContract({
    address: token,
    abi: ERC20_ABI,
    functionName: "balanceOf",
    args: [owner],
  })
}

export async function getAllowance(
  token: Address,
  owner: Address,
  client: BasePublicClient,
): Promise<bigint> {
  return client.readContract({
    address: token,
    abi: ERC20_ABI,
    functionName: "allowance",
    args: [owner, AFI_ADDRESS],
  })
}

export async function assertSufficientBalance(
  token: Address,
  owner: Address,
  required: bigint,
  client: BasePublicClient,
): Promise<void> {
  const balance = await getBalance(token, owner, client)
  if (balance < required) throw new InsufficientBalanceError(token, balance, required)
}

/**
 * Ensures the AFI contract has exactly `amount` allowance for `token`.
 * Returns the approval tx hash, or null if existing allowance was already sufficient.
 */
export async function ensureExactApproval(
  token: Address,
  owner: Address,
  amount: bigint,
  publicClient: BasePublicClient,
  walletClient: BaseWalletClient,
): Promise<string | null> {
  const current = await getAllowance(token, owner, publicClient)
  if (current >= amount) return null

  // Some tokens (USDT-style) reject non-zero → non-zero allowance changes. Reset first.
  if (current > 0n) {
    try {
      const resetHash = await walletClient.writeContract({
        address: token,
        abi: ERC20_ABI,
        functionName: "approve",
        args: [AFI_ADDRESS, 0n],
      })
      await publicClient.waitForTransactionReceipt({ hash: resetHash })
    } catch {
      // Token doesn't require reset, continue
    }
  }

  let hash: `0x${string}`
  try {
    hash = await walletClient.writeContract({
      address: token,
      abi: ERC20_ABI,
      functionName: "approve",
      args: [AFI_ADDRESS, amount],
    })
  } catch (e) {
    throw new ApprovalError((e as Error).message)
  }

  await publicClient.waitForTransactionReceipt({ hash })

  // Verify on-chain before proceeding — protects against race conditions
  const confirmed = await getAllowance(token, owner, publicClient)
  if (confirmed < amount) {
    throw new ApprovalError("allowance not reflected on-chain after confirmation")
  }

  return hash
}
