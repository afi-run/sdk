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
import { AFI_ABI, AFI_ADDRESS, API_BASE_URL } from "./constants.js"
import { NoSignerError, SimulationFailedError } from "./errors.js"
import { fetchTokens } from "./info.js"
import { QuoteBuilder } from "./builder.js"
import { assertSufficientBalance, ensureExactApproval, submitApproval } from "./token.js"
import { simulateSwap, submitSwap as sendSwap } from "./swap.js"
import type { AfiConfig, Address, Hex, Network, PendingSwap, PendingTx, Quote, SwapResult, Token } from "./types.js"
import { NETWORK } from "./types.js"

type BasePublicClient = PublicClient<Transport, typeof base>
type BaseWalletClient = WalletClient<Transport, typeof base, Account>

export class AfiClient {
  private readonly _rpcUrl: string
  private readonly pub: BasePublicClient
  private _apiUrl: string
  private wallet: BaseWalletClient | undefined
  private account: ReturnType<typeof privateKeyToAccount> | undefined

  constructor(config: AfiConfig) {
    this._rpcUrl  = config.rpcUrl
    this._apiUrl  = API_BASE_URL
    this.pub = createPublicClient({ chain: base, transport: http(config.rpcUrl) })
    if (config.privateKey) {
      this.connect(config.privateKey)
    }
  }

  /** The blockchain RPC URL configured on this client. */
  get rpcUrl(): string { return this._rpcUrl }

  /** The full quoter endpoint URL (API base + "/quoter"). */
  get quoterUrl(): string { return this._apiUrl + "/quoter" }

  /** The full info endpoint URL (API base + "/info"). */
  get infoUrl(): string { return this._apiUrl + "/info" }

  /**
   * Changes the base URL used for API calls (default: https://rpc.afi.run).
   * Useful for pointing the SDK at a local or staging instance.
   * Returns `this` for chaining.
   */
  setApiUrl(url: string): this {
    this._apiUrl = url
    return this
  }

  /**
   * Attaches a signer to this client. Returns `this` for chaining.
   */
  connect(privateKey: Hex): this {
    this.account = privateKeyToAccount(privateKey)
    this.wallet  = createWalletClient({
      account:   this.account,
      chain:     base,
      transport: http(this._rpcUrl),
    }) as BaseWalletClient
    return this
  }

  private requireSigner(): { account: ReturnType<typeof privateKeyToAccount>; wallet: BaseWalletClient } {
    if (!this.account || !this.wallet) throw new NoSignerError()
    return { account: this.account, wallet: this.wallet }
  }

  /**
   * Returns tokens available for swapping on the specified network (default: Base).
   */
  async getTokens(network: Network = NETWORK.BASE): Promise<Token[]> {
    return fetchTokens(network, this.infoUrl)
  }

  /** Reads the current protocol fee from the contract (basis points). */
  async getFeeBps(): Promise<number> {
    return this.pub.readContract({
      address:      AFI_ADDRESS,
      abi:          AFI_ABI,
      functionName: "feeBps",
    })
  }

  /**
   * Returns a QuoteBuilder for the given token pair and input amount.
   * Chain methods to configure the quote, then call .get() or .execute().
   *
   * @example
   * const quote = await client
   *   .quote(USDC, WETH, "1000")
   *   .slippage(0.5)
   *   .network(NETWORK.BASE)
   *   .get()
   *
   * @example
   * // Pass Token objects directly from getTokens()
   * const tokens = await client.getTokens()
   * const usdc = tokens.find(t => t.symbol === "USDC")!
   * const result = await client
   *   .quote(usdc, WETH, "1000")
   *   .slippage(0.5)
   *   .execute()
   */
  quote(tokenIn: Address | Token, tokenOut: Address | Token, amountIn: string): QuoteBuilder {
    return new QuoteBuilder(this, tokenIn, tokenOut, amountIn)
  }

  /**
   * Sends an approve tx for exactly `amountWei` of `tokenIn` to the AFI contract.
   * Returns a PendingTx or null if allowance was already sufficient.
   */
  async approve(tokenIn: string, amountWei: bigint): Promise<PendingTx | null> {
    const { account, wallet } = this.requireSigner()
    return submitApproval(
      tokenIn as `0x${string}`,
      account.address,
      amountWei,
      this.pub,
      wallet,
    )
  }

  /**
   * Simulates the swap. Returns true if it would succeed, false otherwise.
   * The optional `log` callback receives the failure reason when simulation fails.
   */
  async simulate(quote: Quote, log?: (reason: string) => void): Promise<boolean> {
    const { account } = this.requireSigner()
    try {
      await simulateSwap(quote, account.address, this.pub)
      return true
    } catch (e) {
      if (e instanceof SimulationFailedError) {
        log?.(e.reason)
        return false
      }
      throw e
    }
  }

  /**
   * Sends the swap tx and returns a PendingSwap without waiting for confirmation.
   */
  async submitSwap(quote: Quote): Promise<PendingSwap> {
    const { account, wallet } = this.requireSigner()
    return sendSwap(quote, account.address, this.pub, wallet)
  }

  /**
   * Executes a swap from a pre-fetched quote.
   *
   * Flow: balance check → approve (exact, waits) → simulate → submitSwap → wait
   */
  async executeSwap(quote: Quote): Promise<SwapResult> {
    const { account, wallet } = this.requireSigner()
    await assertSufficientBalance(quote.tokenIn, account.address, quote.amountInWei, this.pub)
    await ensureExactApproval(quote.tokenIn, account.address, quote.amountInWei, this.pub, wallet)

    let simulationReason: string | undefined
    const ok = await this.simulate(quote, (r) => { simulationReason = r })
    if (!ok) throw new SimulationFailedError(simulationReason ?? "simulation failed")

    const pending = await sendSwap(quote, account.address, this.pub, wallet)
    return pending.wait()
  }
}
