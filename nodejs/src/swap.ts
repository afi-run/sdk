import { parseEventLogs } from "viem"
import type { Account, PublicClient, Transport, WalletClient } from "viem"
import type { base } from "viem/chains"
import { AFI_ABI, AFI_ADDRESS } from "./constants.js"
import { SimulationFailedError, SwapRevertedError } from "./errors.js"
import type { Address, Hex, Quote, SwapResult } from "./types.js"

type BasePublicClient = PublicClient<Transport, typeof base>
type BaseWalletClient = WalletClient<Transport, typeof base, Account>

export async function simulate(
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
    const err = e as { shortMessage?: string; message?: string; data?: string }
    throw new SimulationFailedError(
      err.shortMessage ?? err.message ?? "unknown revert",
      err.data,
    )
  }
}

export async function executeSwap(
  quote: Quote,
  sender: Address,
  publicClient: BasePublicClient,
  walletClient: BaseWalletClient,
): Promise<SwapResult> {
  const gas = await publicClient.estimateContractGas({
    address: AFI_ADDRESS,
    abi: AFI_ABI,
    functionName: "swap",
    args: [quote.tokenIn, quote.amountInWei, quote.tokenOut, quote.minOutWei, quote.steps],
    account: sender,
  })

  let hash: Hex
  try {
    hash = await walletClient.writeContract({
      address: AFI_ADDRESS,
      abi: AFI_ABI,
      functionName: "swap",
      args: [quote.tokenIn, quote.amountInWei, quote.tokenOut, quote.minOutWei, quote.steps],
      gas: (gas * 12n) / 10n,
    })
  } catch (e) {
    throw new SwapRevertedError((e as Error).message)
  }

  const receipt = await publicClient.waitForTransactionReceipt({ hash })

  const logs = parseEventLogs({
    abi: AFI_ABI,
    eventName: "SwapExecuted",
    logs: receipt.logs,
  })

  const event = logs[0]
  if (!event) throw new SwapRevertedError("SwapExecuted event not found in receipt")

  return {
    txHash: hash,
    blockNumber: receipt.blockNumber,
    amountIn: event.args.amountIn,
    amountOut: event.args.amountOut,
    tokenIn: event.args.assetIn,
    tokenOut: event.args.assetOut,
    gasUsed: receipt.gasUsed,
  }
}
