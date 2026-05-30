import { parseEventLogs, type TransactionReceipt } from "viem"
import type { Account, PublicClient, Transport, WalletClient } from "viem"
import type { base } from "viem/chains"
import { AFI_ABI, AFI_ADDRESS } from "./constants.js"
import { decodeRevertReason, describeDecodedRevert } from "./decode.js"
import { SimulationFailedError, SwapRevertedError } from "./errors.js"
import type { Address, Hex, PendingSwap, Quote, SwapResult, WaitForTxOptions } from "./types.js"
import { formatUnits } from "./utils.js"

/** Computes the on-chain fee from a receipt-shaped object. */
export function feeFromReceipt(gasUsed: bigint, effectiveGasPrice: bigint | undefined): {
  effectiveGasPrice: bigint
  feeWei: bigint
  feeEth: string
} {
  const eff = effectiveGasPrice ?? 0n
  const wei = gasUsed * eff
  return { effectiveGasPrice: eff, feeWei: wei, feeEth: formatUnits(wei, 18) }
}

/**
 * Parses a transaction receipt for the AFI `SwapExecuted` event and reconstructs the SwapResult.
 * Returns `null` when no SwapExecuted log is present (e.g. the tx wasn't an AFI swap).
 *
 * Use this for hashes obtained outside the SDK (indexers, replay tools, queued jobs).
 */
export function parseSwapResult(receipt: TransactionReceipt): SwapResult | null {
  const logs = parseEventLogs({ abi: AFI_ABI, eventName: "SwapExecuted", logs: receipt.logs })
  const event = logs[0]
  if (!event) return null
  const fee = feeFromReceipt(receipt.gasUsed, (receipt as { effectiveGasPrice?: bigint }).effectiveGasPrice)
  return {
    txHash:      receipt.transactionHash,
    blockNumber: receipt.blockNumber,
    amountIn:    event.args.amountIn,
    amountOut:   event.args.amountOut,
    tokenIn:     event.args.assetIn,
    tokenOut:    event.args.assetOut,
    gasUsed:     receipt.gasUsed,
    ...fee,
  }
}

type BasePublicClient = PublicClient<Transport, typeof base>
type BaseWalletClient = WalletClient<Transport, typeof base, Account>

export async function simulateSwap(
  quote: Quote,
  sender: Address,
  client: BasePublicClient,
): Promise<void> {
  try {
    await client.simulateContract({
      address: AFI_ADDRESS,
      abi: AFI_ABI,
      functionName: "swap",
      args: [quote.tokenIn, quote.amountInWei, quote.tokenOut, quote.minOutWei, quote.steps],
      account: sender,
    })
  } catch (e: unknown) {
    const err = e as { shortMessage?: string; message?: string; data?: string; cause?: { data?: string } }
    // viem nests the raw revert hex in different places depending on the path.
    const rawData = err.data ?? err.cause?.data
    const decoded = decodeRevertReason(rawData)
    const reason = decoded
      ? describeDecodedRevert(decoded)
      : err.shortMessage ?? err.message ?? "unknown revert"
    throw new SimulationFailedError(reason, rawData, decoded ?? undefined)
  }
}

/** Returns gas * (1 + bufferPercent/100). bufferPercent <= 0 returns gas unchanged. */
export function applyGasBuffer(gas: bigint, bufferPercent: number): bigint {
  if (bufferPercent <= 0) return gas
  return (gas * BigInt(100 + Math.floor(bufferPercent))) / 100n
}

export async function submitSwap(
  quote: Quote,
  sender: Address,
  publicClient: BasePublicClient,
  walletClient: BaseWalletClient,
  gasBufferPercent: number,
  nonce?: number,
): Promise<PendingSwap> {
  let gas: bigint
  try {
    gas = await publicClient.estimateContractGas({
      address: AFI_ADDRESS,
      abi: AFI_ABI,
      functionName: "swap",
      args: [quote.tokenIn, quote.amountInWei, quote.tokenOut, quote.minOutWei, quote.steps],
      account: sender,
    })
  } catch (e) {
    // estimateGas reverts swallow the contract message — replay via simulate to surface it.
    try {
      await simulateSwap(quote, sender, publicClient)
    } catch (simErr) {
      if (simErr instanceof SimulationFailedError) {
        throw new SwapRevertedError(`estimate gas: ${simErr.reason}`)
      }
    }
    throw new SwapRevertedError(`estimate gas: ${(e as Error).message}`)
  }

  let hash: Hex
  try {
    hash = await walletClient.writeContract({
      address: AFI_ADDRESS,
      abi: AFI_ABI,
      functionName: "swap",
      args: [quote.tokenIn, quote.amountInWei, quote.tokenOut, quote.minOutWei, quote.steps],
      gas: applyGasBuffer(gas, gasBufferPercent),
      ...(nonce !== undefined ? { nonce } : {}),
    })
  } catch (e) {
    const err = e as { message?: string; data?: string; cause?: { data?: string } }
    const decoded = decodeRevertReason(err.data ?? err.cause?.data)
    throw new SwapRevertedError(decoded ? describeDecodedRevert(decoded) : (e as Error).message, decoded ?? undefined)
  }

  return {
    txHash: hash,
    wait: async (opts?: WaitForTxOptions): Promise<SwapResult> => {
      const receipt = await publicClient.waitForTransactionReceipt({
        hash,
        confirmations: opts?.confirmations,
        timeout: opts?.timeoutMs,
      })
      const result = parseSwapResult(receipt)
      if (!result) throw new SwapRevertedError("SwapExecuted event not found in receipt")
      return result // parseSwapResult already attaches fee from the receipt
    },
  }
}
