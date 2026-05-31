import type { Account, PublicClient, Transport, WalletClient } from "viem"
import type { base } from "viem/chains"
import { AFI_ADDRESS, ERC20_ABI } from "./constants.js"
import { ApprovalError, InsufficientBalanceError } from "./errors.js"
import { fetchTokenInfo } from "./multicall.js"
import { applyGasBuffer, feeFromReceipt } from "./swap.js"
import type { Address, PendingTx, TxReceipt, WaitForTxOptions } from "./types.js"

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

/** Reads `allowance(owner, spender)` against an arbitrary spender. */
export async function getAllowanceFor(
  token: Address,
  owner: Address,
  spender: Address,
  client: BasePublicClient,
): Promise<bigint> {
  return client.readContract({
    address: token,
    abi: ERC20_ABI,
    functionName: "allowance",
    args: [owner, spender],
  })
}

export async function assertSufficientBalance(
  token: Address,
  owner: Address,
  required: bigint,
  client: BasePublicClient,
): Promise<void> {
  const balance = await getBalance(token, owner, client)
  if (balance >= required) return
  // Enrich the error with symbol/decimals via multicall — one extra RPC on the failure path
  // gives a vastly better message ("Insufficient USDC: have 0.5, need 1") than raw addresses.
  const info = await fetchTokenInfo(token, client).catch(() => undefined)
  throw new InsufficientBalanceError(token, balance, required, owner, info?.symbol, info?.decimals)
}

export async function writeApprove(
  token: Address,
  spender: Address,
  amount: bigint,
  publicClient: BasePublicClient,
  walletClient: BaseWalletClient,
  account: Address,
  gasBufferPercent: number,
  nonce?: number,
): Promise<`0x${string}`> {
  const nonceParam = nonce !== undefined ? { nonce } : {}
  let gas: bigint
  try {
    gas = await publicClient.estimateContractGas({
      address: token,
      abi: ERC20_ABI,
      functionName: "approve",
      args: [spender, amount],
      account,
    })
  } catch {
    // Fall back to letting writeContract estimate again (some RPCs differ between estimate paths).
    return walletClient.writeContract({
      address: token,
      abi: ERC20_ABI,
      functionName: "approve",
      args: [spender, amount],
      ...nonceParam,
    })
  }
  return walletClient.writeContract({
    address: token,
    abi: ERC20_ABI,
    functionName: "approve",
    args: [spender, amount],
    gas: applyGasBuffer(gas, gasBufferPercent),
    ...nonceParam,
  })
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
  gasBufferPercent: number,
): Promise<string | null> {
  const current = await getAllowance(token, owner, publicClient)
  if (current >= amount) return null

  // Some tokens (USDT-style) reject non-zero → non-zero allowance changes. Reset first.
  // Preserve the reset error so it can be surfaced if the subsequent approve also fails.
  let resetErr: Error | undefined
  if (current > 0n) {
    try {
      const resetHash = await writeApprove(token, AFI_ADDRESS, 0n, publicClient, walletClient, owner, gasBufferPercent)
      await publicClient.waitForTransactionReceipt({ hash: resetHash })
    } catch (e) {
      resetErr = e as Error
    }
  }

  let hash: `0x${string}`
  try {
    hash = await writeApprove(token, AFI_ADDRESS, amount, publicClient, walletClient, owner, gasBufferPercent)
  } catch (e) {
    const msg = (e as Error).message
    throw new ApprovalError(resetErr ? `${msg} (allowance reset also failed: ${resetErr.message})` : msg)
  }

  await publicClient.waitForTransactionReceipt({ hash })

  // Verify on-chain before proceeding — protects against race conditions
  const confirmed = await getAllowance(token, owner, publicClient)
  if (confirmed < amount) {
    throw new ApprovalError("allowance not reflected on-chain after confirmation")
  }

  return hash
}

/**
 * Sends the approve tx and returns a PendingTx without waiting for confirmation.
 * Returns null if existing allowance is already sufficient.
 * The USDT-style reset pre-step still waits internally before sending the real approval.
 */
export async function submitApproval(
  token: Address,
  owner: Address,
  amount: bigint,
  publicClient: BasePublicClient,
  walletClient: BaseWalletClient,
  gasBufferPercent: number,
  nonce?: number,
): Promise<PendingTx | null> {
  const current = await getAllowance(token, owner, publicClient)
  if (current >= amount) return null

  let resetErr: Error | undefined
  let curNonce = nonce
  if (current > 0n) {
    try {
      const resetHash = await writeApprove(token, AFI_ADDRESS, 0n, publicClient, walletClient, owner, gasBufferPercent, curNonce)
      await publicClient.waitForTransactionReceipt({ hash: resetHash })
      if (curNonce !== undefined) curNonce += 1
    } catch (e) { resetErr = e as Error }
  }

  let hash: `0x${string}`
  try {
    hash = await writeApprove(token, AFI_ADDRESS, amount, publicClient, walletClient, owner, gasBufferPercent, curNonce)
  } catch (e) {
    const msg = (e as Error).message
    throw new ApprovalError(resetErr ? `${msg} (allowance reset also failed: ${resetErr.message})` : msg)
  }

  return {
    txHash: hash,
    wait: async (opts?: WaitForTxOptions): Promise<TxReceipt> => {
      const receipt = await publicClient.waitForTransactionReceipt({
        hash,
        confirmations: opts?.confirmations,
        timeout: opts?.timeoutMs,
      })
      const confirmed = await getAllowance(token, owner, publicClient)
      if (confirmed < amount) throw new ApprovalError("allowance not reflected on-chain after confirmation")
      const fee = feeFromReceipt(receipt.gasUsed, (receipt as { effectiveGasPrice?: bigint }).effectiveGasPrice)
      return { blockNumber: receipt.blockNumber, gasUsed: receipt.gasUsed, ...fee }
    },
  }
}
