import {
  createPublicClient,
  createWalletClient,
  http,
  type Account,
  type PublicClient,
  type Transport,
  type WalletClient,
} from "viem"
import { privateKeyToAccount } from "viem/accounts"
import { base } from "viem/chains"
import { AFI_ABI, AFI_ADDRESS } from "./constants.js"
import { fetchTokens } from "./info.js"
import { fetchQuote } from "./quoter.js"
import { assertSufficientBalance, ensureExactApproval, getDecimals } from "./token.js"
import { simulate, executeSwap } from "./swap.js"
import type { AfiConfig, Quote, SwapParams, SwapResult, Token } from "./types.js"

type BasePublicClient = PublicClient<Transport, typeof base>
type BaseWalletClient = WalletClient<Transport, typeof base, Account>

export class AfiClient {
  private readonly pub: BasePublicClient
  private readonly wallet: BaseWalletClient
  private readonly account: ReturnType<typeof privateKeyToAccount>
  private readonly rpcUrl: string

  constructor(config: AfiConfig) {
    this.rpcUrl = config.rpcUrl
    this.account = privateKeyToAccount(config.privateKey)
    this.pub = createPublicClient({ chain: base, transport: http(config.rpcUrl) })
    this.wallet = createWalletClient({
      account: this.account,
      chain: base,
      transport: http(config.rpcUrl),
    }) as BaseWalletClient
  }

  /**
   * Returns all tokens available for swapping on Base.
   * Use this to discover supported tokens before building a swap.
   */
  async getTokens(): Promise<Token[]> {
    return fetchTokens()
  }

  /** Reads the current protocol fee from the contract (basis points). */
  async getFeeBps(): Promise<number> {
    return this.pub.readContract({
      address: AFI_ADDRESS,
      abi: AFI_ABI,
      functionName: "feeBps",
    })
  }

  /**
   * Fetches a quote for the given swap params.
   * Returns pricing, route path, minimum output, and the encoded steps
   * needed for execution. No on-chain interaction — safe to call freely.
   */
  async getQuote(params: SwapParams): Promise<Quote> {
    const [decimals, feeBps] = await Promise.all([
      getDecimals(params.tokenIn, this.pub),
      this.getFeeBps(),
    ])
    return fetchQuote(params, decimals, feeBps, this.rpcUrl)
  }

  /**
   * Approves exactly `amountWei` of `tokenIn` to the AFI contract.
   * Returns the tx hash, or null if existing allowance was already sufficient.
   * Called automatically by executeSwap() — only use this directly for custom flows.
   */
  async approve(tokenIn: string, amountWei: bigint): Promise<string | null> {
    return ensureExactApproval(
      tokenIn as `0x${string}`,
      this.account.address,
      amountWei,
      this.pub,
      this.wallet,
    )
  }

  /**
   * Executes a swap from a pre-fetched quote.
   *
   * Flow: balance check → approve (exact) → simulate → swap
   *
   * Use this after reviewing a quote from getQuote().
   * Throws SimulationFailedError before sending any tx if the swap would revert.
   */
  async executeSwap(quote: Quote): Promise<SwapResult> {
    await assertSufficientBalance(quote.tokenIn, this.account.address, quote.amountInWei, this.pub)
    await this.approve(quote.tokenIn, quote.amountInWei)
    await simulate(quote, this.account.address, this.pub)
    return executeSwap(quote, this.account.address, this.pub, this.wallet)
  }

  /**
   * Convenience method: runs the full flow in one call.
   * Equivalent to: const quote = await getQuote(params); return executeSwap(quote)
   */
  async swap(params: SwapParams): Promise<SwapResult> {
    const quote = await this.getQuote(params)
    return this.executeSwap(quote)
  }
}
