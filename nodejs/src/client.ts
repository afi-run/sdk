import {
  createPublicClient,
  createWalletClient,
  encodeFunctionData,
  http,
  type Abi,
  type Account,
  type PublicClient,
  type Transport,
  type TransactionReceipt,
  type WalletClient,
} from "viem"
import { privateKeyToAccount } from "viem/accounts"
import { base } from "viem/chains"
import {
  AFI_ABI,
  AFI_ADDRESS,
  AFI_ADDRESSES,
  API_BASE_URL,
  BASE_CHAIN_ID,
  DEFAULT_GAS_BUFFER_PERCENT,
  ERC20_ABI,
  NETWORK_CHAIN_IDS,
  OWNABLE2STEP_ABI,
  ROUTE_REGISTRY_ABI,
  TREASURY_OWNER,
} from "./constants.js"
import { ApprovalError, NoSignerError } from "./errors.js"
import { addressUrl, txUrl } from "./explorer.js"
import {
  encodeApprove,
  encodeBatchSwapFor,
  encodeRevoke,
  encodeSwap,
  encodeSwapFor,
  type EncodedTx,
  type SwapForArgs,
} from "./encode.js"
import {
  encodeAfiAddRule,
  encodeAfiClearRules,
  encodeAfiClearUserFeeBps,
  encodeAfiPause,
  encodeAfiRescueTokens,
  encodeAfiResetAnyUserOverride,
  encodeAfiSetFeeBps,
  encodeAfiSetOperator,
  encodeAfiSetTreasury,
  encodeAfiSetUserFeeBps,
  encodeAfiSetUserFeeBpsBatch,
  encodeAfiUnpause,
} from "./afi-admin.js"
import { ZERO_ADDRESS } from "./address.js"
import {
  findArbitrage,
  findPath,
  getLiquidationCandidates,
  getRoutes,
  liquidate,
  priceQuote,
  quoteDex,
  type ArbitrageRequest,
  type RouteQuote,
  type DexAction,
  type DexQuoteRequest,
  type LiquidateRequest,
  type LiquidationResult,
  type AavePosition,
  type LiquidationCandidatesRequest,
  type PathRequest,
  type PathQuote,
  type PriceQuoteRequest,
  type RoutesRequest,
  type Route,
} from "./quoter.js"
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
import { assertSufficientBalance, ensureExactApproval, getAllowance, getAllowanceFor, getBalance, submitApproval, writeApprove } from "./token.js"
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
  private _nonceLock: Promise<void> = Promise.resolve()

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
      const nonce = await this.allocateNonce(opts?.nonce)
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

      const nonce = await this.allocateNonce(opts?.nonce)
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
      const nonce = await this.allocateNonce(opts?.nonce)
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
      const nonce = await this.allocateNonce(opts?.nonce)
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

  /**
   * @internal — allocates the nonce for the next write tx.
   *
   * Serializes parallel allocations via a Promise queue so concurrent callers
   * never receive the same nonce. When managed mode is off, the override path
   * still returns synchronously through this awaitable wrapper.
   */
  private async allocateNonce(override?: number): Promise<number | undefined> {
    if (override !== undefined) return override
    if (!this._managedNonce) return undefined

    let release!: () => void
    const next = new Promise<void>((resolve) => { release = resolve })
    const prev = this._nonceLock
    this._nonceLock = next
    await prev
    try {
      if (this._localNonce === null) {
        const { account } = this.requireSigner()
        this._localNonce = await this.pub.getTransactionCount({
          address: account.address,
          blockTag: "pending",
        })
      }
      const allocated = this._localNonce
      this._localNonce = allocated + 1
      return allocated
    } finally {
      release()
    }
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

  // ─── Generic write helper ────────────────────────────────────────────────────

  /**
   * Generic write helper. Estimates gas, applies the SDK gas buffer, allocates
   * the next managed nonce (when enabled), submits the tx and waits for
   * `opts.confirmations` confirmations (default: 1).
   *
   * Throws NoSignerError when no signer is attached. Pre-encoded calls (`{to, data}`)
   * should use the wallet client directly; this helper exists for typed contract calls.
   */
  async sendContractTx<TAbi extends Abi>(
    to: Address,
    abi: TAbi,
    functionName: string,
    args: readonly unknown[],
    opts?: { value?: bigint; confirmations?: number; nonce?: number },
  ): Promise<TransactionReceipt> {
    const { account, wallet } = this.requireSigner()
    let gas: bigint
    try {
      gas = await this.pub.estimateContractGas({
        address: to,
        abi: abi as Abi,
        functionName,
        args,
        account: account.address,
        ...(opts?.value !== undefined ? { value: opts.value } : {}),
      } as Parameters<typeof this.pub.estimateContractGas>[0])
    } catch (estErr) {
      // Re-issue as eth_call so the RPC echoes the revert reason — the bare
      // estimateContractGas message frequently strips the inner error.
      try {
        const data = encodeFunctionData({
          abi: abi as Abi,
          functionName,
          args,
        })
        await this.pub.call({
          account: account.address,
          to,
          data,
          ...(opts?.value !== undefined ? { value: opts.value } : {}),
        })
      } catch (callErr) {
        const shortMessage = (callErr as { shortMessage?: string }).shortMessage
        const inner = shortMessage ?? (callErr as Error).message
        throw new Error(`sendContractTx: ${functionName} would revert: ${inner}`)
      }
      // pub.call did not throw — surface the original estimate error.
      throw new Error(`estimateContractGas(${functionName}): ${(estErr as Error).message}`)
    }
    const gasWithBuffer = (gas * BigInt(100 + Math.max(0, Math.floor(this._gasBufferPercent)))) / 100n
    const nonce = await this.allocateNonce(opts?.nonce)
    const hash = await wallet.writeContract({
      address: to,
      abi: abi as Abi,
      functionName,
      args,
      gas: gasWithBuffer,
      ...(opts?.value !== undefined ? { value: opts.value } : {}),
      ...(nonce !== undefined ? { nonce } : {}),
    } as Parameters<typeof wallet.writeContract>[0])
    return this.pub.waitForTransactionReceipt({
      hash,
      confirmations: opts?.confirmations ?? 1,
    })
  }

  // ─── Operator workflows ──────────────────────────────────────────────────────

  /**
   * Operator-only. Executes `swapFor(user, …)` on the AFI router. The user must
   * have already approved the router for `amountIn` of `tokenIn`.
   *
   * When `precheck` is true (default), reads `ERC20(tokenIn).allowance(user, AFI)`
   * off-chain and throws `ApprovalError` before submitting any tx if it is
   * below `amountIn`. Pass `precheck: false` to skip the read.
   */
  async swapFor(args: {
    user: Address
    tokenIn: Address
    tokenOut: Address
    amountIn: bigint
    minOut: bigint
    steps: Hex
    precheck?: boolean
  }): Promise<TransactionReceipt> {
    if (args.precheck !== false) {
      const allowance = await getAllowanceFor(args.tokenIn, args.user, AFI_ADDRESS, this.pub)
      if (allowance < args.amountIn) {
        throw new ApprovalError(
          `user ${args.user} allowance (${allowance.toString()}) on token ${args.tokenIn} ` +
          `for AFI ${AFI_ADDRESS} is below amountIn (${args.amountIn.toString()})`,
        )
      }
    }
    return this.sendContractTx(AFI_ADDRESS, AFI_ABI, "swapFor", [
      args.user,
      args.tokenIn,
      args.amountIn,
      args.tokenOut,
      args.minOut,
      args.steps,
    ])
  }

  /** Operator-only. Executes the AFI `batchSwapFor(tuple[])` entry point. */
  async batchSwapFor(swaps: readonly SwapForArgs[]): Promise<TransactionReceipt> {
    const tuples = swaps.map((s) => ({
      user: s.user,
      tokenIn: s.tokenIn,
      amountIn: s.amountInWei,
      tokenOut: s.tokenOut,
      minOut: s.minOutWei,
      params: s.steps,
    }))
    return this.sendContractTx(AFI_ADDRESS, AFI_ABI, "batchSwapFor", [tuples])
  }

  // ─── Admin (owner-only) ──────────────────────────────────────────────────────

  /** Owner-only. `pause()` on the AFI router. */
  async adminPause(): Promise<TransactionReceipt> {
    return this.sendContractTx(AFI_ADDRESS, AFI_ABI, "pause", [])
  }

  /** Owner-only. `unpause()` on the AFI router. */
  async adminUnpause(): Promise<TransactionReceipt> {
    return this.sendContractTx(AFI_ADDRESS, AFI_ABI, "unpause", [])
  }

  /** Owner-only. `setTreasury(addr)` — `addr` cannot be the zero address. */
  async adminSetTreasury(addr: Address): Promise<TransactionReceipt> {
    if (addr.toLowerCase() === ZERO_ADDRESS) throw new Error("treasury cannot be the zero address")
    return this.sendContractTx(AFI_ADDRESS, AFI_ABI, "setTreasury", [addr])
  }

  /** Owner-only. `setFeeBps(bps)` — `bps` must be in [0, 50]. */
  async adminSetFeeBps(bps: number): Promise<TransactionReceipt> {
    // Re-uses encodeAfiSetFeeBps's range check via a side call.
    encodeAfiSetFeeBps(bps)
    return this.sendContractTx(AFI_ADDRESS, AFI_ABI, "setFeeBps", [bps])
  }

  /** Owner-only. `setUserFeeBps(user, bps)` — `bps` must be in [0, 50]. */
  async adminSetUserFeeBps(user: Address, bps: number): Promise<TransactionReceipt> {
    encodeAfiSetUserFeeBps(user, bps)
    return this.sendContractTx(AFI_ADDRESS, AFI_ABI, "setUserFeeBps", [user, bps])
  }

  /** Owner-only. `setUserFeeBpsBatch(users, bps)` — lengths must match. */
  async adminSetUserFeeBpsBatch(users: Address[], bps: number[]): Promise<TransactionReceipt> {
    encodeAfiSetUserFeeBpsBatch(users, bps)
    return this.sendContractTx(AFI_ADDRESS, AFI_ABI, "setUserFeeBpsBatch", [users, bps])
  }

  /** Owner-only. `clearUserFeeBps(user)`. */
  async adminClearUserFeeBps(user: Address): Promise<TransactionReceipt> {
    return this.sendContractTx(AFI_ADDRESS, AFI_ABI, "clearUserFeeBps", [user])
  }

  /** Owner-only. `resetAnyUserOverride()`. */
  async adminResetAnyUserOverride(): Promise<TransactionReceipt> {
    return this.sendContractTx(AFI_ADDRESS, AFI_ABI, "resetAnyUserOverride", [])
  }

  /** Owner-only. `addRule(rule)` — rejects the zero address client-side. */
  async adminAddRule(rule: Address): Promise<TransactionReceipt> {
    if (rule.toLowerCase() === ZERO_ADDRESS) throw new Error("rule cannot be the zero address")
    return this.sendContractTx(AFI_ADDRESS, AFI_ABI, "addRule", [rule])
  }

  /** Owner-only. `clearRules()`. */
  async adminClearRules(): Promise<TransactionReceipt> {
    return this.sendContractTx(AFI_ADDRESS, AFI_ABI, "clearRules", [])
  }

  /** Owner-only. `setOperator(op, value)`. */
  async adminSetOperator(op: Address, value: boolean): Promise<TransactionReceipt> {
    return this.sendContractTx(AFI_ADDRESS, AFI_ABI, "setOperator", [op, value])
  }

  /** Owner-only. `rescueTokens(token, value, to)`. */
  async adminRescueTokens(token: Address, value: bigint, to: Address): Promise<TransactionReceipt> {
    return this.sendContractTx(AFI_ADDRESS, AFI_ABI, "rescueTokens", [token, value, to])
  }

  // ─── Ownable2Step handover ───────────────────────────────────────────────────

  /**
   * Calls `acceptOwnership()` on any Ownable2Step contract. The connected
   * signer must already be the contract's `pendingOwner()` — otherwise the tx
   * will revert with `OwnableUnauthorizedAccount`.
   */
  async acceptOwnership(
    contractAddr: Address,
    opts?: { confirmations?: number },
  ): Promise<TransactionReceipt> {
    return this.sendContractTx(contractAddr, OWNABLE2STEP_ABI, "acceptOwnership", [], opts)
  }

  /**
   * Sequentially calls `acceptOwnership()` on every contract in `contracts`.
   * Order is preserved and txs are submitted one-at-a-time (not via
   * `Promise.all`) so the managed nonce stays deterministic on flaky RPCs.
   */
  async acceptOwnershipBatch(
    contracts: Address[],
    opts?: { confirmations?: number },
  ): Promise<TransactionReceipt[]> {
    const receipts: TransactionReceipt[] = []
    for (const addr of contracts) {
      receipts.push(await this.acceptOwnership(addr, opts))
    }
    return receipts
  }

  // ─── Read helpers ────────────────────────────────────────────────────────────

  private afiAddress(chainId?: number): Address {
    if (chainId === undefined) return AFI_ADDRESS
    const a = AFI_ADDRESSES[chainId]
    if (!a) throw new Error(`no AFI deployment for chainId=${chainId}`)
    return a
  }


  /** Reads the AFI router pause flag. */
  async isPaused(chainId?: number): Promise<boolean> {
    return this.pub.readContract({
      address: this.afiAddress(chainId),
      abi: AFI_ABI,
      functionName: "paused",
    })
  }

  /** Reads `feeBpsOf(user)` — falls back to the default fee when no override is set. */
  async getFeeBpsOf(user: Address, chainId?: number): Promise<number> {
    return this.pub.readContract({
      address: this.afiAddress(chainId),
      abi: AFI_ABI,
      functionName: "feeBpsOf",
      args: [user],
    })
  }

  /** Reads `hasRules()` — true when at least one rule contract is registered. */
  async hasRules(chainId?: number): Promise<boolean> {
    return this.pub.readContract({
      address: this.afiAddress(chainId),
      abi: AFI_ABI,
      functionName: "hasRules",
    })
  }

  /** Reads the configured treasury address on the AFI router. */
  async getTreasuryAddress(chainId?: number): Promise<Address> {
    return this.pub.readContract({
      address: this.afiAddress(chainId),
      abi: AFI_ABI,
      functionName: "treasury",
    })
  }

  /** Reads the immutable RouteRegistry the AFI router was deployed with. */
  async getRegistryAddress(chainId?: number): Promise<Address> {
    return this.pub.readContract({
      address: this.afiAddress(chainId),
      abi: AFI_ABI,
      functionName: "registry",
    })
  }

  /** Reads the AFI router's primaryOperator slot. */
  async getPrimaryOperator(chainId?: number): Promise<Address> {
    return this.pub.readContract({
      address: this.afiAddress(chainId),
      abi: AFI_ABI,
      functionName: "primaryOperator",
    })
  }

  /** Returns true when `addr` is set as an AFI operator (or matches primaryOperator). */
  async isAfiOperator(addr: Address, chainId?: number): Promise<boolean> {
    return this.pub.readContract({
      address: this.afiAddress(chainId),
      abi: AFI_ABI,
      functionName: "isOperator",
      args: [addr],
    })
  }

  /** Reads `Ownable2Step.owner()` on the AFI router. */
  async getOwner(chainId?: number): Promise<Address> {
    return this.pub.readContract({
      address: this.afiAddress(chainId),
      abi: AFI_ABI,
      functionName: "owner",
    })
  }

  /** Reads `Ownable2Step.pendingOwner()` on the AFI router. */
  async getPendingOwner(chainId?: number): Promise<Address> {
    return this.pub.readContract({
      address: this.afiAddress(chainId),
      abi: AFI_ABI,
      functionName: "pendingOwner",
    })
  }

  /** Looks up a single route in the RouteRegistry by uint16 ID. */
  async getRoute(id: number, chainId?: number): Promise<Address> {
    const registry = await this.getRegistryAddress(chainId)
    return this.pub.readContract({
      address: registry,
      abi: ROUTE_REGISTRY_ABI,
      functionName: "getRoute",
      args: [id],
    })
  }

  /**
   * Lists all currently-registered routes (IDs 1..9 per DeployInfra) in a
   * single multicall round-trip. Skips any that revert (route not registered).
   */
  async listRoutes(chainId?: number): Promise<Map<number, Address>> {
    const registry = await this.getRegistryAddress(chainId)
    const ids = [1, 2, 3, 4, 5, 6, 7, 8, 9]
    const results = await this.multicall(
      ids.map((id) => ({
        address: registry,
        abi: ROUTE_REGISTRY_ABI,
        functionName: "getRoute",
        args: [id],
      })),
      { allowFailure: true },
    )
    const out = new Map<number, Address>()
    results.forEach((r, idx) => {
      if (r.status === "success") {
        const addr = r.result as Address
        if (addr && addr.toLowerCase() !== ZERO_ADDRESS) out.set(ids[idx], addr)
      }
    })
    return out
  }

  /** ERC20.balanceOf on the AFI router's treasury for `token`. */
  async getTreasuryBalance(token: Address, chainId?: number): Promise<bigint> {
    const treasury = await this.getTreasuryAddress(chainId)
    return this.pub.readContract({
      address: token,
      abi: ERC20_ABI,
      functionName: "balanceOf",
      args: [treasury],
    })
  }

  /**
   * Bundles deployment-health checks into a single multicall:
   *
   *   paused, treasury, registry, primaryOperator, owner, hasRules, pendingOwner
   *
   * Returns `{ ok, issues }`. `ok=false` when treasury/registry/primaryOperator
   * is the zero address, paused is true, `pendingOwner` is set (handover not
   * complete) or `treasury` differs from the `TREASURY_OWNER` constant.
   */
  async verifyDeployment(chainId?: number): Promise<{ ok: boolean; issues: string[] }> {
    const afi = this.afiAddress(chainId)
    const results = await this.multicall([
      { address: afi, abi: AFI_ABI, functionName: "paused" },
      { address: afi, abi: AFI_ABI, functionName: "treasury" },
      { address: afi, abi: AFI_ABI, functionName: "registry" },
      { address: afi, abi: AFI_ABI, functionName: "primaryOperator" },
      { address: afi, abi: AFI_ABI, functionName: "owner" },
      { address: afi, abi: AFI_ABI, functionName: "hasRules" },
      { address: afi, abi: AFI_ABI, functionName: "pendingOwner" },
    ], { allowFailure: true })

    const issues: string[] = []
    const need = (label: string, idx: number): unknown => {
      const r = results[idx]
      if (r.status !== "success") {
        issues.push(`${label}() reverted: ${r.error.message}`)
        return undefined
      }
      return r.result
    }
    const paused = need("paused", 0) as boolean | undefined
    const treasury = need("treasury", 1) as Address | undefined
    const registry = need("registry", 2) as Address | undefined
    const primary = need("primaryOperator", 3) as Address | undefined
    need("owner", 4)
    need("hasRules", 5)
    const pendingOwner = need("pendingOwner", 6) as Address | undefined

    if (paused === true) issues.push("router is paused")
    if (treasury && treasury.toLowerCase() === ZERO_ADDRESS) issues.push("treasury is zero address")
    if (registry && registry.toLowerCase() === ZERO_ADDRESS) issues.push("registry is zero address")
    if (primary && primary.toLowerCase() === ZERO_ADDRESS) issues.push("primaryOperator is zero address")
    if (pendingOwner && pendingOwner.toLowerCase() !== ZERO_ADDRESS) {
      issues.push(`ownership transfer pending: ${pendingOwner}`)
    }
    // Only flag a mismatch when TREASURY_OWNER is configured (non-zero).
    if (
      treasury &&
      TREASURY_OWNER.toLowerCase() !== ZERO_ADDRESS &&
      treasury.toLowerCase() !== TREASURY_OWNER.toLowerCase()
    ) {
      issues.push(`treasury mismatch: on-chain ${treasury} vs expected ${TREASURY_OWNER}`)
    }

    return { ok: issues.length === 0, issues }
  }

  // ─── afi-rpc HTTP endpoints ──────────────────────────────────────────────────

  /** Posts `/arbitrage` and returns the candidate routes for `tokenIn`. */
  async findArbitrage(req: ArbitrageRequest): Promise<RouteQuote[]> {
    return this.logged("findArbitrage", () => findArbitrage(this._apiUrl, req), "api")
  }

  /** Posts `/command {action:"path"}` — priced multi-hop route for an explicit path. */
  async findPath(req: PathRequest): Promise<PathQuote> {
    return this.logged("findPath", () => findPath(this._apiUrl, req), "api")
  }

  /** Posts `/command {action:"routes"}` — candidate token paths for a pair. */
  async getRoutes(req: RoutesRequest): Promise<Route[]> {
    return this.logged("getRoutes", () => getRoutes(this._apiUrl, req), "api")
  }

  /** Posts `/aave` — current liquidation candidates on Aave V3 for `network`. */
  async getLiquidationCandidates(
    req: LiquidationCandidatesRequest,
  ): Promise<AavePosition[]> {
    return this.logged("getLiquidationCandidates", () => getLiquidationCandidates(this._apiUrl, req), "api")
  }

  /** Posts `/liquidation-call` — quotes a single liquidationCall for the given user. */
  async liquidate(req: LiquidateRequest): Promise<LiquidationResult> {
    return this.logged("liquidate", () => liquidate(this._apiUrl, req), "api")
  }

  /** Posts `/command {action:"price"}` — per-DEX quotes for the pair. */
  async priceQuote(req: PriceQuoteRequest): Promise<RouteQuote[]> {
    return this.logged("priceQuote", () => priceQuote(this._apiUrl, req), "api")
  }

  /** Posts `/command {action:<dex>}` — single-DEX quote helper. */
  async quoteDex(dex: DexAction, req: DexQuoteRequest): Promise<RouteQuote[]> {
    return this.logged(`quoteDex:${dex}`, () => quoteDex(this._apiUrl, dex, req), "api")
  }
}
