import type { AfiClient } from "./client.js"
import { fetchQuote, type QuoteRequest } from "./quoter.js"
import type { Address, Dex, Network, Quote, RpcUrlInfo, SwapResult, Token } from "./types.js"
import { NETWORK } from "./types.js"

function resolveAddress(t: Address | Token): Address {
  return typeof t === "string" ? t : t.address
}

export class QuoteBuilder {
  private req: QuoteRequest

  constructor(
    private readonly client: AfiClient,
    tokenIn: Address | Token,
    tokenOut: Address | Token,
    amountIn: string,
  ) {
    this.req = {
      tokenIn:  resolveAddress(tokenIn),
      tokenOut: resolveAddress(tokenOut),
      amountIn,
      slippage: 0.5,
      maxHops:  2,
      network:  NETWORK.BASE,
    }
  }

  /** Sets slippage tolerance percentage (default: 0.5). */
  slippage(v: number): this { this.req.slippage = v; return this }

  /** Sets the maximum number of route hops (default: 2). */
  maxHops(n: number): this { this.req.maxHops = n; return this }

  /** Sets the target network (default: NETWORK.BASE). */
  network(n: Network): this { this.req.network = n; return this }

  /** Sets the base asset for price fields. Populates tokenInBasePrice and tokenOutBasePrice in the Quote. */
  priceBase(base: string): this { this.req.priceBase = base; return this }

  /** Restricts routing to the specified DEX protocols. */
  dexs(...dexs: Dex[]): this { this.req.dexs = dexs; return this }

  /** Sets a specific block number for the quote (default: "latest"). */
  blockNumber(n: string | number): this { this.req.blockNumber = n; return this }

  /** Sets custom RPC endpoints for the quoter API. */
  rpcUrls(...urls: RpcUrlInfo[]): this { this.req.rpcUrls = urls; return this }

  /** Fetches the quote. No transaction is sent. */
  async get(): Promise<Quote> {
    return this.client._runLogged("getQuote", async () => {
      const feeBps = await this.client.getFeeBps()
      const req = { ...this.req }
      if (!req.rpcUrls || req.rpcUrls.length === 0) {
        req.rpcUrls = [{ url: this.client.rpcUrl }]
      }
      return fetchQuote(req, feeBps, this.client.quoterUrl)
    })
  }

  /** Fetches the quote and executes the swap. Requires a connected signer. */
  async execute(): Promise<SwapResult> {
    const quote = await this.get()
    return this.client.executeSwap(quote)
  }
}
