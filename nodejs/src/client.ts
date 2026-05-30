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
import { AFI_ABI, AFI_ADDRESS, API_BASE_URL, DEFAULT_GAS_BUFFER_PERCENT, NETWORK_CHAIN_IDS } from "./constants.js"
import { ApprovalError, NoSignerError } from "./errors.js"
import { addressUrl, txUrl } from "./explorer.js"
import { encodeApprove, encodeRevoke, encodeSwap, type EncodedTx } from "./encode.js"
import { fetchTokens } from "./info.js"
import {
  fetchTokenInfo,
  fetchTokenInfoBatch,
  genericMulticall,
  type MulticallContract,
  type MulticallResult,
  type TokenMetadata,
} from "./multicall.js"
import { QuoteBuilder } from "./builder.js"
import { assertSufficientBalance, ensureExactApproval, getAllowance, getBalance, submitApproval, writeApprove } from "./token.js"
import { feeFromReceipt, simulateSwap, submitSwap as sendSwap } from "./swap.js"
import { isSimulationFailedError } from "./errors.js"
import type {
  AfiConfig,
  Address,
  ExecuteOptions,
  HealthCheck,
  Hex,
  LogEvent,
  Logger,
  Network,
  PendingSwap,
  PendingTx,
  PreflightProblem,
  PreflightReport,
  Quote,
  SwapCostEstimate,
  SwapResult,
  Token,
  TokenInfo,
  TokenPrice,
  TxReceipt,
  TxStatus,
  WaitForTxOptions,
} from "./types.js"
import { NETWORK } from "./types.js"

type BasePublicClient = PublicClient<Transport, typeof base>
type BaseWalletClient = WalletClient<Transport, typeof base, Account>

export class AfiClient {
  private readonly _rpcUrl: string
  private readonly pub: BasePublicClient
  private _apiUrl: string
  private _gasBufferPercent: number
  private wallet: BaseWalletClient | undefined
  private account: ReturnType<typeof privateKeyToAccount> | undefined
  private tokensCache: Map<Network, Promise<Token[]>> = new Map()
  private metaCache: Map<string, TokenMetadata> = new Map()
  private _logger: Logger | undefined
  private _chainIdPromise: Promise<number> | undefined
  private _managedNonce = false
  private _localNonce: number | null = null

  constructor(config: AfiConfig) {
    this._rpcUrl  = config.rpcUrl
    this._apiUrl  = API_BASE_URL
    this._gasBufferPercent = config.gasBufferPercent ?? DEFAULT_GAS_BUFFER_PERCENT
    this._logger = config.logger
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

  /** Percentage added on top of estimated gas for approve/swap txs. */
  get gasBufferPercent(): number { return this._gasBufferPercent }

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
   * Overrides the default gas buffer percentage applied on top of the estimated gas
   * for write txs (approve, swap). Pass 0 to disable the buffer. Returns `this` for chaining.
   */
  setGasBufferPercent(percent: number): this {
    this._gasBufferPercent = percent
    return this
  }

  /** Attaches (or replaces) the diagnostic logger. Pass `undefined` to disable. */
  setLogger(logger: Logger | undefined): this {
    this._logger = logger
    return this
  }

  /**
   * Reads the RPC's chain ID (cached after first call). Use to verify the configured
   * RPC matches the network you expect.
   */
  async chainId(): Promise<number> {
    if (!this._chainIdPromise) {
      this._chainIdPromise = this.pub.getChainId().catch((e) => {
        this._chainIdPromise = undefined
        throw e
      })
    }
    return this._chainIdPromise
  }

  /**
   * Returns the Network the RPC is connected to, or `null` if the chain ID doesn't
   * match any known network in NETWORK_CHAIN_IDS.
   */
  async detectNetwork(): Promise<Network | null> {
    const id = await this.chainId()
    const match = Object.entries(NETWORK_CHAIN_IDS).find(([, cid]) => cid === id)
    return (match?.[0] as Network) ?? null
  }

  /**
   * Polls until the transaction `hash` reaches the requested number of confirmations
   * (default: 1). Use this for hashes obtained outside the SDK (e.g. persisted from a prior run).
   */
  async waitForTx(hash: Hex, opts?: WaitForTxOptions): Promise<TxReceipt> {
    const receipt = await this.pub.waitForTransactionReceipt({
      hash,
      confirmations: opts?.confirmations,
      timeout: opts?.timeoutMs,
    })
    const fee = feeFromReceipt(receipt.gasUsed, (receipt as { effectiveGasPrice?: bigint }).effectiveGasPrice)
    return { blockNumber: receipt.blockNumber, gasUsed: receipt.gasUsed, ...fee }
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

  private async logged<T>(method: string, fn: () => Promise<T>, kind: LogEvent["kind"] = "rpc"): Promise<T> {
    if (!this._logger) return fn()
    const start = Date.now()
    try {
      const result = await fn()
      this._logger({ kind, method, durationMs: Date.now() - start, ok: true })
      return result
    } catch (e) {
      this._logger({ kind, method, durationMs: Date.now() - start, ok: false, error: e })
      throw e
    }
  }

  /** @internal — used by QuoteBuilder to plumb the logger into get(). */
  async _runLogged<T>(method: string, fn: () => Promise<T>, kind: LogEvent["kind"] = "api"): Promise<T> {
    return this.logged(method, fn, kind)
  }

  private resolveOwner(owner?: Address): Address {
    if (owner) return owner
    const { account } = this.requireSigner()
    return account.address
  }

  /**
   * Returns tokens available for swapping on the specified network (default: Base).
   * Results are cached per-network — call `clearTokensCache()` to force a refresh.
   */
  async getTokens(network: Network = NETWORK.BASE): Promise<Token[]> {
    const cached = this.tokensCache.get(network)
    if (cached) return cached
    const promise = fetchTokens(network, this.infoUrl).catch((err) => {
      this.tokensCache.delete(network)
      throw err
    })
    this.tokensCache.set(network, promise)
    return promise
  }

  /** Invalidates the in-memory `getTokens()` cache (all networks, or just one). */
  clearTokensCache(network?: Network): void {
    if (network) this.tokensCache.delete(network)
    else this.tokensCache.clear()
  }

  /**
   * Looks up a token by symbol on `network` (default: Base). Case-insensitive.
   * Uses `getTokens()`'s cache, so calling this in a hot loop is cheap.
   * Returns `null` when no active token matches.
   */
  async findToken(symbol: string, network: Network = NETWORK.BASE): Promise<Token | null> {
    const target = symbol.toLowerCase()
    const tokens = await this.getTokens(network)
    return tokens.find((t) => t.symbol.toLowerCase() === target) ?? null
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
   * Fetches symbol, name, decimals — plus balance and allowance against the AFI contract
   * when `owner` is provided — in a single multicall.
   *
   * Pass no owner (or omit it) to fetch metadata only. Pass `"self"` to use the connected wallet.
   */
  async tokenInfo(token: Address, owner?: Address | "self"): Promise<TokenInfo> {
    const ownerAddr = owner === "self" ? this.resolveOwner() : owner
    return fetchTokenInfo(token, this.pub, ownerAddr, this.metaCache)
  }

  /**
   * Batched version of `tokenInfo()`: fetches metadata (+ optional balance/allowance) for
   * N tokens in a single multicall round-trip. Order of the result matches the input.
   * Reuses the in-memory metadata cache, so repeat calls only fetch live balance/allowance.
   */
  async tokenInfoBatch(tokens: Address[], owner?: Address | "self"): Promise<TokenInfo[]> {
    const ownerAddr = owner === "self" ? this.resolveOwner() : owner
    return fetchTokenInfoBatch(tokens, this.pub, ownerAddr, this.metaCache)
  }

  /** Clears the in-memory `(symbol, name, decimals)` cache used by `tokenInfo`. */
  clearTokenMetadataCache(): void { this.metaCache.clear() }

  /**
   * Generic Multicall3 wrapper. Bundles N read calls into a single RPC round-trip.
   * Use for arbitrary contract reads beyond what the SDK exposes (pool prices,
   * custom DEX state, your own contracts).
   *
   * @example
   * const [a, b] = await client.multicall([
   *   { address: tokenA, abi: ERC20_ABI, functionName: "totalSupply" },
   *   { address: tokenB, abi: ERC20_ABI, functionName: "totalSupply" },
   * ])
   * if (a.status === "success") console.log("totalSupply A:", a.result)
   */
  async multicall(
    contracts: MulticallContract[],
    opts?: { allowFailure?: boolean },
  ): Promise<MulticallResult[]> {
    return genericMulticall(this.pub, contracts, opts)
  }

  /** Native ETH balance of `owner` (defaults to the connected wallet). */
  async getEthBalance(owner?: Address): Promise<bigint> {
    return this.pub.getBalance({ address: this.resolveOwner(owner) })
  }

  /** Explorer URL for a transaction hash (default network: Base). */
  txUrl(hash: string, network: Network = NETWORK.BASE): string {
    return txUrl(hash, network)
  }

  /** Explorer URL for an address (default network: Base). */
  addressUrl(address: string, network: Network = NETWORK.BASE): string {
    return addressUrl(address, network)
  }

  /**
   * Reads how much `token` the AFI contract is allowed to spend on behalf of owner.
   * Owner defaults to the connected wallet (requires a signer).
   */
  async getAllowance(token: Address, owner?: Address): Promise<bigint> {
    return getAllowance(token, this.resolveOwner(owner), this.pub)
  }

  /**
   * Returns true when the AFI contract already has at least `amountWei` of allowance.
   * Use this to skip `approve()` when not needed.
   */
  async hasAllowance(token: Address, amountWei: bigint, owner?: Address): Promise<boolean> {
    const current = await this.getAllowance(token, owner)
    return current >= amountWei
  }

  /** ERC20 balance of `owner` (defaults to the connected wallet). */
  async getBalance(token: Address, owner?: Address): Promise<bigint> {
    return getBalance(token, this.resolveOwner(owner), this.pub)
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
  async approve(tokenIn: Address, amountWei: bigint, opts?: { nonce?: number }): Promise<PendingTx | null> {
    return this.logged("approve", async () => {
      const { account, wallet } = this.requireSigner()
      const nonce = this.allocateNonce(opts?.nonce)
      return submitApproval(tokenIn, account.address, amountWei, this.pub, wallet, this._gasBufferPercent, nonce)
    })
  }

  /**
   * Sends `approve(AFI, 0)` for `token`, zeroing the router's allowance. Returns
   * null when the allowance is already zero. Use as a post-swap security cleanup.
   */
  async revoke(token: Address, opts?: { nonce?: number }): Promise<PendingTx | null> {
    return this.logged("revoke", async () => {
      const { account, wallet } = this.requireSigner()
      const current = await getAllowance(token, account.address, this.pub)
      if (current === 0n) return null

      const nonce = this.allocateNonce(opts?.nonce)
      let hash: Hex
      try {
        hash = await writeApprove(token, AFI_ADDRESS, 0n, this.pub, wallet, account.address, this._gasBufferPercent, nonce)
      } catch (e) {
        throw new ApprovalError((e as Error).message)
      }
      return {
        txHash: hash,
        wait: async (waitOpts?) => {
          const receipt = await this.pub.waitForTransactionReceipt({
            hash,
            confirmations: waitOpts?.confirmations,
            timeout: waitOpts?.timeoutMs,
          })
          const confirmed = await getAllowance(token, account.address, this.pub)
          if (confirmed !== 0n) throw new ApprovalError("allowance not zeroed after confirmation")
          const fee = feeFromReceipt(receipt.gasUsed, (receipt as { effectiveGasPrice?: bigint }).effectiveGasPrice)
          return { blockNumber: receipt.blockNumber, gasUsed: receipt.gasUsed, ...fee }
        },
      }
    })
  }

  /**
   * Re-fetches a quote using the same logical request as a stale one. All
   * parameters (network, slippage, maxHops, priceBase, dexs) are reused —
   * only the live route, prices and timestamps are refreshed.
   *
   *     if (isQuoteStale(quote, 30)) quote = await client.refreshQuote(quote)
   */
  async refreshQuote(quote: Quote): Promise<Quote> {
    const b = this.quote(quote.tokenIn, quote.tokenOut, quote.amountIn)
      .slippage(quote.slippage)
      .network(quote.network)
      .maxHops(quote.maxHops)
    if (quote.priceBase) b.priceBase(quote.priceBase)
    if (quote.dexs && quote.dexs.length > 0) b.dexs(...quote.dexs)
    return b.get()
  }

  /** Pre-encoded `{to, data, value}` for the swap, ready to feed any external signer. */
  encodeSwap(quote: Quote): EncodedTx { return encodeSwap(quote) }

  /** Pre-encoded `approve(AFI, amount)` tx for `token`. */
  encodeApprove(token: Address, amountWei: bigint): EncodedTx { return encodeApprove(token, amountWei) }

  /** Pre-encoded `approve(AFI, 0)` tx for `token` (revoke). */
  encodeRevoke(token: Address): EncodedTx { return encodeRevoke(token) }

  /**
   * Simulates the swap via eth_call.
   * Resolves on success; throws SimulationFailedError with the revert reason on failure.
   */
  async simulate(quote: Quote): Promise<void> {
    return this.logged("simulate", async () => {
      const { account } = this.requireSigner()
      await simulateSwap(quote, account.address, this.pub)
    })
  }

  /**
   * Sends the swap tx and returns a PendingSwap without waiting for confirmation.
   */
  async submitSwap(quote: Quote, opts?: { nonce?: number }): Promise<PendingSwap> {
    return this.logged("submitSwap", async () => {
      const { account, wallet } = this.requireSigner()
      const nonce = this.allocateNonce(opts?.nonce)
      return sendSwap(quote, account.address, this.pub, wallet, this._gasBufferPercent, nonce)
    })
  }

  /**
   * Executes a swap from a pre-fetched quote.
   *
   * Flow: balance check → approve (exact, waits) → simulate → submitSwap → wait
   *
   * Pass `opts` to require more than 1 confirmation or set a timeout.
   */
  async executeSwap(quote: Quote, opts?: ExecuteOptions): Promise<SwapResult> {
    return this.logged("executeSwap", async () => {
      const { account, wallet } = this.requireSigner()
      await assertSufficientBalance(quote.tokenIn, account.address, quote.amountInWei, this.pub)
      await ensureExactApproval(
        quote.tokenIn,
        account.address,
        quote.amountInWei,
        this.pub,
        wallet,
        this._gasBufferPercent,
      )
      await simulateSwap(quote, account.address, this.pub)
      const nonce = this.allocateNonce(opts?.nonce)
      const pending = await sendSwap(quote, account.address, this.pub, wallet, this._gasBufferPercent, nonce)
      return pending.wait({ confirmations: opts?.confirmations, timeoutMs: opts?.timeoutMs })
    })
  }

  /**
   * Projects the gas cost of executing the quote (without sending a tx).
   * Requires a signer because eth_estimateGas needs a `from` address.
   */
  async estimateSwapCost(quote: Quote): Promise<SwapCostEstimate> {
    const { account } = this.requireSigner()
    const gas = await this.pub.estimateContractGas({
      address: AFI_ADDRESS,
      abi: AFI_ABI,
      functionName: "swap",
      args: [quote.tokenIn, quote.amountInWei, quote.tokenOut, quote.minOutWei, quote.steps],
      account: account.address,
    })
    const gasWithBuffer = (gas * BigInt(100 + Math.max(0, Math.floor(this._gasBufferPercent)))) / 100n

    const [block, tip] = await Promise.all([
      this.pub.getBlock({ blockTag: "latest" }),
      this.pub.estimateMaxPriorityFeePerGas().catch(() => 0n),
    ])
    const baseFee = block.baseFeePerGas ?? 0n
    const gasPriceWei = baseFee * 2n + tip
    const totalWei = gasWithBuffer * gasPriceWei

    const { formatUnits } = await import("./utils.js")
    return {
      gas,
      gasWithBuffer,
      gasPriceWei,
      totalWei,
      totalEth: formatUnits(totalWei, 18),
    }
  }

  /**
   * Probes the RPC (chainId) and the AFI API (`/info?network=base`) in parallel
   * and reports per-endpoint status. Useful at startup to fail fast.
   */
  async health(): Promise<HealthCheck> {
    const probeRpc = async () => {
      const start = Date.now()
      try {
        const id = await this.pub.getChainId()
        return { ok: true, durationMs: Date.now() - start, detail: `chainId=${id}` }
      } catch (e) {
        return { ok: false, durationMs: Date.now() - start, error: e }
      }
    }
    const probeApi = async () => {
      const start = Date.now()
      try {
        const res = await fetch(`${this.infoUrl}?network=${NETWORK.BASE}`)
        if (!res.ok) {
          return { ok: false, durationMs: Date.now() - start, detail: `HTTP ${res.status}` }
        }
        return { ok: true, durationMs: Date.now() - start, detail: "ok" }
      } catch (e) {
        return { ok: false, durationMs: Date.now() - start, error: e }
      }
    }
    const [rpc, api] = await Promise.all([probeRpc(), probeApi()])
    return { rpc, api }
  }

  // ─── Nonce management ────────────────────────────────────────────────────────

  /** Reads the pending nonce of the connected wallet directly from the RPC. */
  async getNonce(): Promise<number> {
    const { account } = this.requireSigner()
    return this.pub.getTransactionCount({ address: account.address, blockTag: "pending" })
  }

  /**
   * Enables managed-nonce mode. The SDK fetches the current pending nonce once
   * and then maintains a local counter that is incremented atomically for each
   * write tx. Use this for bots that submit multiple swaps in parallel without
   * waiting for confirmations between them.
   */
  async useManagedNonce(): Promise<void> {
    this._localNonce = await this.getNonce()
    this._managedNonce = true
  }

  /** Disables managed-nonce mode. Subsequent writes will query the RPC each time. */
  disableManagedNonce(): void {
    this._managedNonce = false
    this._localNonce = null
  }

  /** Re-syncs the local managed-nonce counter from the chain. */
  async resetManagedNonce(): Promise<void> {
    if (!this._managedNonce) return
    this._localNonce = await this.getNonce()
  }

  /** @internal — allocates the nonce for the next write tx. */
  private allocateNonce(override?: number): number | undefined {
    if (override !== undefined) return override
    if (!this._managedNonce || this._localNonce === null) return undefined
    const n = this._localNonce
    this._localNonce = n + 1
    return n
  }

  // ─── Tx introspection ────────────────────────────────────────────────────────

  /**
   * Non-blocking status check for an arbitrary tx hash. Returns immediately
   * with one of `"pending" | "success" | "failed" | "unknown"`.
   */
  async getTxStatus(hash: Hex): Promise<TxStatus> {
    try {
      const receipt = await this.pub.getTransactionReceipt({ hash })
      return receipt.status === "success" ? "success" : "failed"
    } catch {
      try {
        await this.pub.getTransaction({ hash })
        return "pending"
      } catch {
        return "unknown"
      }
    }
  }

  // ─── Token pricing ───────────────────────────────────────────────────────────

  /**
   * Light-weight price lookup for a token pair via the quoter. Returns the
   * current exchange rate without committing to the rest of the swap flow.
   *
   * @example
   *   const { price } = await client.getTokenPrice(USDC, WETH)
   *   console.log(`1 USDC = ${price} WETH`)
   */
  async getTokenPrice(
    tokenIn: Address | Token,
    tokenOut: Address | Token,
    opts?: { network?: Network; amount?: string; slippage?: number },
  ): Promise<TokenPrice> {
    const q = await this.quote(tokenIn, tokenOut, opts?.amount ?? "1")
      .slippage(opts?.slippage ?? 0.5)
      .network(opts?.network ?? NETWORK.BASE)
      .get()
    return { price: q.tokenInPrice, inverse: q.tokenOutPrice }
  }

  // ─── Preflight ───────────────────────────────────────────────────────────────

  /**
   * Combined balance + allowance + simulation check that runs without sending
   * any transaction. Use it to drive a UI "ready to swap" indicator without
   * forcing the user through approve.
   */
  async preflight(quote: Quote): Promise<PreflightReport> {
    const { account } = this.requireSigner()
    const [balance, allowance] = await Promise.all([
      this.getBalance(quote.tokenIn, account.address),
      this.getAllowance(quote.tokenIn, account.address),
    ])

    const problems: PreflightProblem[] = []
    if (balance < quote.amountInWei) {
      problems.push({
        code: "INSUFFICIENT_BALANCE",
        message: `have ${balance.toString()}, need ${quote.amountInWei.toString()}`,
      })
    }

    const needsApproval = allowance < quote.amountInWei

    // Only simulate when balance and allowance are sufficient — otherwise the
    // revert is expected and would just duplicate INSUFFICIENT_BALANCE noise.
    if (balance >= quote.amountInWei && !needsApproval) {
      try {
        await simulateSwap(quote, account.address, this.pub)
      } catch (e) {
        if (isSimulationFailedError(e)) {
          problems.push({ code: "SIMULATION_FAILED", message: e.reason })
        } else {
          throw e
        }
      }
    }

    return {
      canExecute: problems.length === 0,
      needsApproval,
      problems,
      balance,
      allowance,
    }
  }
}
