# AFI SDK

> Production-grade SDK for executing token swaps on EVM chains via the [AFI Protocol](https://afi.run).

Build swap interfaces, trading bots, analytics tools and indexers without
reimplementing route discovery, slippage math, allowance flows, gas buffering,
revert decoding or event parsing.

| | |
|---|---|
| **Languages** | TypeScript (Node.js 18+) · Go 1.21+ |
| **Networks (quote)** | Base · BSC · Arbitrum · Ethereum · Unichain |
| **Networks (execute)** | All of the above (chain ID detected from the RPC) |
| **Translations** | [Português (BR)](./README.pt-BR.md) · [Español](./README.es.md) |
| **License** | MIT |

---

## Why this SDK

- **One-call swaps** — `client.swap()` chains quote → balance check → approve → simulate → submit → wait.
- **Staged flow** — every step is also exposed individually for granular UI control.
- **Safe by default** — exact allowances, on-chain `minOut` enforcement, simulation before broadcast.
- **Multi-chain quotes** — request a quote from 5 EVM chains via a single client.
- **Operational ergonomics** — health probes, structured logs, JSON serialization, multicall reads, configurable gas buffer, confirmations, timeouts.

---

## What do you want to do?

A quick map from your role to the entrypoint you want. Every entry on the right
links to a section of this document; the encoders ship today, the higher-level
`client.*` wrappers are sugar on top of them.

| Role | Goal | Use |
|---|---|---|
| End user | Swap your own tokens | `client.swap()` or `client.quote().execute()` |
| Operator | Swap for 1 pre-approved user | `client.swapFor({ user, tokenIn, tokenOut, amountIn })` |
| Operator | Batch swap for many users | `client.batchSwapFor([{ user, ... }, ...])` |
| Owner | Pause / unpause router | `client.adminPause()` / `client.adminUnpause()` |
| Owner | Change global fee | `client.adminSetFeeBps(bps)` |
| Owner | Per-user fee override | `client.adminSetUserFeeBps(user, bps)` |
| Owner | Add validation rule | `client.adminAddRule(rule)` |
| Inspector | Pre-flight check | `client.verifyDeployment(chainId)` |
| Indexer | Parse events | `parseSwapExecuted(logs)`, `parseFeeCollected(logs)`, ... |

---

## Table of contents

- [Requirements](#requirements)
- [Installation](#installation)
- [Quick start](#quick-start)
- [Core concepts](#core-concepts)
- [API reference](#api-reference)
  - [Client construction](#client-construction)
  - [Read operations](#read-operations)
  - [Quote builder](#quote-builder)
  - [Write operations (require a signer)](#write-operations-require-a-signer)
  - [Transaction utilities](#transaction-utilities)
  - [Configuration](#configuration)
- [Helpers](#helpers)
- [Logging & diagnostics](#logging--diagnostics)
- [Error handling](#error-handling)
- [Security model](#security-model)
- [Recipes](#recipes)
- [Operator workflows](#operator-workflows)
- [Admin / governance](#admin--governance)
- [Event indexing](#event-indexing)
- [Per-DEX step builders](#per-dex-step-builders)
- [HTTP quoter endpoints](#http-quoter-endpoints)
- [Migration guide](#migration-guide)
- [Networks & constants](#networks--constants)
- [Examples directory](#examples-directory)
- [Development](#development)
- [License](#license)

---

## Requirements

| Runtime    | Minimum  | Recommended |
|------------|----------|-------------|
| Node.js    | 18.x     | 20.x LTS    |
| TypeScript | 5.0      | latest      |
| Go         | 1.21     | 1.22+       |

You also need an HTTP RPC endpoint for each chain you intend to read from or
execute on. Public providers (Ankr, Alchemy, Infura, drpc, …) work for
development; **use a paid tier (or your own node) for production workloads**
to avoid quoter timeouts and rate-limit reverts.

---

## Installation

### TypeScript / Node.js

```bash
npm install @afi-run/sdk     # or: pnpm add @afi-run/sdk · yarn add @afi-run/sdk
```

Until the package ships on npm:

```bash
git clone https://github.com/afi-run/sdk.git
npm install ./sdk/nodejs
```

### Go

```bash
go get github.com/afi-run/sdk/go
```

```go
import afi "github.com/afi-run/sdk/go"
```

---

## Quick start

### TypeScript — read-only quote

```typescript
import { AfiClient, NETWORK, formatUnits } from "@afi-run/sdk"

const client = new AfiClient({
  rpcUrl: "https://rpc.ankr.com/base/YOUR_API_KEY",
})

const quote = await client
  .quote(
    "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",  // USDC
    "0x4200000000000000000000000000000000000006",  // WETH
    "1000",
  )
  .slippage(0.5)
  .network(NETWORK.BASE)
  .get()

console.log(`Estimated:  ~${quote.amountOut} WETH`)
console.log(`Min out:    ${quote.minOut} WETH`)
console.log(`Route hops: ${quote.hops.length}`)
console.log(`Created:    ${new Date(quote.createdAt).toISOString()}`)
```

### TypeScript — one-shot swap

```typescript
client.connect("0xYOUR_PRIVATE_KEY")

const result = await client
  .quote(USDC, WETH, "500")
  .slippage(0.5)
  .execute({ confirmations: 1 })

console.log(`Tx:        ${client.txUrl(result.txHash)}`)
console.log(`Received:  ${formatUnits(result.amountOut, 18)} WETH`)
console.log(`Gas used:  ${result.gasUsed}`)
```

### Go — read-only quote

```go
import (
    "context"
    "fmt"
    "log"

    afi "github.com/afi-run/sdk/go"
    "github.com/ethereum/go-ethereum/common"
)

func main() {
    client, err := afi.NewClient(afi.Config{
        RPCURL: "https://rpc.ankr.com/base/YOUR_API_KEY",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()
    usdc := common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913")

    quote, err := client.GetQuote(ctx,
        afi.From(usdc, afi.WETH, "1000"),
        afi.WithSlippage(0.5),
        afi.OnNetwork(afi.NetworkBase),
    )
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Estimated:  ~%s WETH\n", quote.AmountOut)
    fmt.Printf("Min out:    %s WETH\n", quote.MinOut)
    fmt.Printf("Route hops: %d\n", len(quote.Hops))
}
```

### Go — one-shot swap

```go
client.Connect("YOUR_PRIVATE_KEY")

result, err := client.Swap(ctx,
    afi.From(usdc, afi.WETH, "500"),
    afi.WithSlippage(0.5),
)
if err != nil {
    log.Fatal(err)
}
url, _ := client.TxURL(result.TxHash.Hex())
fmt.Printf("Tx:       %s\n", url)
fmt.Printf("Received: %s WETH\n", afi.FormatUnits(result.AmountOut, 18))
```

---

## Core concepts

### The swap lifecycle

Every swap goes through five stages. `executeSwap(quote)` runs stages 2–5
atomically; each stage is also exposed individually for staged UI flows.

```
1. Quote          ─ POST /quoter — compute the route, slippage, minOut
2. Balance check  ─ ERC20.balanceOf(owner) ≥ amountIn
3. Approve        ─ ERC20.approve(AFI, amountInWei)        (skipped if allowance is enough)
4. Simulate       ─ eth_call AFI.swap(...)                 (fails fast on revert)
5. Submit + wait  ─ broadcast and wait for confirmations
```

### Read-only mode vs signer mode

The client has two modes selected by whether a private key is configured:

- **Read-only** — `quote`, `tokenInfo`, `getBalance`, `getEthBalance`,
  `getAllowance`, `hasAllowance`, `getFeeBps`, `chainId`, `detectNetwork`,
  `health`, `txUrl`, `addressUrl`.
- **Signer mode** (adds) — `approve`, `simulate`, `submitSwap`, `executeSwap`,
  `swap`, `estimateSwapCost`.

Read-only methods remain available after `connect()`. Call `requireSigner`
internally to enforce signer presence on write methods; you'll get a
`NoSignerError` if a private key is missing.

### Gas buffer model

All write transactions (approve + swap) multiply the result of `eth_estimateGas`
by `(1 + gasBufferPercent / 100)`. Default is **+15 %**. Configure with
`gasBufferPercent` in the client config, or override at runtime with
`setGasBufferPercent(n)`. Pass `0` to disable the buffer entirely.

The same value is applied to the gas the SDK passes to `writeContract` /
`SendTransaction`, never to the price (maxFeePerGas is computed as
`baseFee * 2 + tip` regardless).

### Slippage and `minOut` guarantee

Every `Quote` carries `minOutWei` — the minimum output the AFI router will
accept on-chain. The contract reverts execution if the actual output would be
lower, so users can never receive less than this value. The SDK refuses to
build quotes with `minOutWei = 0` for safety.

Slippage is expressed in percent (`0.5` = 0.5%) and applied by the quoter.
Compute your own `minOut` with the [`calculateMinOut`](#slippage-calculator)
helper if you need to derive it client-side.

### Builder pattern vs functional options

- **TypeScript** — `client.quote(...)` returns a fluent `QuoteBuilder`.
  Chain `.slippage()`, `.maxHops()`, `.network()`, etc., then call `.get()` or `.execute()`.
- **Go** — `client.GetQuote(ctx, opts...)` takes functional options (`afi.From`,
  `afi.WithSlippage`, `afi.OnNetwork`, …).

Both surface the same configuration; pick what your codebase already prefers.

---

## API reference

### Client construction

#### TypeScript

```typescript
new AfiClient(config: AfiConfig)

interface AfiConfig {
  rpcUrl:             string             // required — RPC endpoint for the execution chain
  privateKey?:        Hex                // optional — enables signer mode
  gasBufferPercent?:  number             // default: 15 — percentage added on top of estimateGas
  logger?:            Logger             // optional — diagnostic callback
}
```

#### Go

```go
afi.NewClient(cfg afi.Config) (*afi.Client, error)

type Config struct {
    RPCURL           string  // required
    PrivateKey       string  // optional — hex with or without 0x prefix
    GasBufferPercent uint    // default: 15 — zero means use the default; SetGasBufferPercent(0) disables
    Logger           Logger  // optional — diagnostic callback
}
```

`Close()` (Go) / no equivalent (TS) closes the underlying RPC connection.

---

### Read operations

| Method | Returns | Description |
|---|---|---|
| `getTokens(network?)` / `GetTokens(ctx, network?)` | `Token[]` | List active tokens. Cached per-network. |
| `findToken(symbol, network?)` / `FindToken(ctx, symbol, network?)` | `Token \| null` | Case-insensitive symbol lookup. Uses the cache. |
| `clearTokensCache(network?)` / `ClearTokensCache(network?)` | `void` | Invalidate the cache (all networks or just one). |
| `getFeeBps()` / `GetFeeBps(ctx)` | `number` / `uint16` | Current protocol fee from the AFI contract. |
| `tokenInfo(token, owner?)` / `TokenInfo(ctx, token, owner)` | `TokenInfo` | symbol/name/decimals (+ balance/allowance) in **one multicall**. |
| `tokenInfoBatch(tokens, owner?)` / `TokenInfoBatch(ctx, tokens, owner)` | `TokenInfo[]` | Same as above for N tokens in a single multicall. |
| `getBalance(token, owner?)` / `GetBalance(ctx, token, owner?)` | `bigint` / `*big.Int` | ERC-20 balance of `owner`. |
| `getEthBalance(owner?)` / `GetETHBalance(ctx, owner?)` | `bigint` / `*big.Int` | Native ETH balance. |
| `getAllowance(token, owner?)` / `GetAllowance(ctx, token, owner?)` | `bigint` / `*big.Int` | How much the AFI router can spend on behalf of `owner`. |
| `hasAllowance(token, amount, owner?)` / `HasAllowance(ctx, token, amount, owner?)` | `boolean` | Convenience: `getAllowance >= amount`. |
| `chainId()` / `ChainID(ctx)` | `number` / `*big.Int` | Chain ID reported by the RPC (cached on first call). |
| `detectNetwork()` / `DetectNetwork(ctx)` | `Network \| null` | Maps the chain ID to a known `Network`. |
| `health()` / `Health(ctx)` | `HealthCheck` | Parallel RPC + API liveness probe. |
| `estimateSwapCost(quote)` / `EstimateSwapCost(ctx, quote)` | `SwapCostEstimate` | Project gas cost without sending a tx. **Requires signer.** |

`owner` parameters default to the connected wallet when omitted. In Go, pass
`common.Address{}` to mean "use the connected wallet". `TokenInfo` accepts
`"self"` (TS) as a shorthand for the same intent.

#### Token

```typescript
interface Token {
  address:  Address     // 0x… 20-byte address
  symbol:   string      // e.g. "USDC"
  decimals: number      // e.g. 6
  active:   boolean     // false ⇒ deprecated/paused token
}
```

#### TokenInfo

```typescript
interface TokenInfo {
  address:    Address
  symbol:     string
  name:       string
  decimals:   number
  owner?:     Address    // present only when an owner was provided
  balance?:   bigint     // ERC-20 balance of owner
  allowance?: bigint     // allowance granted to AFI by owner
}
```

#### HealthCheck

```typescript
interface HealthEndpoint {
  ok:          boolean
  durationMs:  number
  detail?:     string    // "chainId=8453" for RPC, "ok" or "HTTP 503" for API
  error?:      unknown
}

interface HealthCheck {
  rpc: HealthEndpoint
  api: HealthEndpoint
}
```

#### SwapCostEstimate

```typescript
interface SwapCostEstimate {
  gas:           bigint   // raw eth_estimateGas result
  gasWithBuffer: bigint   // gas * (1 + gasBufferPercent/100)
  gasPriceWei:   bigint   // maxFeePerGas the SDK would use = baseFee * 2 + tip
  totalWei:      bigint   // gasWithBuffer * gasPriceWei
  totalEth:      string   // totalWei formatted as ETH (18 decimals)
}
```

---

### Quote builder

#### TypeScript

```typescript
client.quote(tokenIn: Address | Token, tokenOut: Address | Token, amountIn: string): QuoteBuilder
```

| Method            | Default     | Description |
|-------------------|-------------|-------------|
| `.slippage(v)`    | `0.5`       | Slippage tolerance (percent) |
| `.maxHops(n)`     | `2`         | Max route hops |
| `.network(n)`     | `BASE`      | Target network |
| `.priceBase(s)`   | —           | Sets `tokenInBasePrice` / `tokenOutBasePrice` on the result |
| `.dexs(...)`      | —           | Restrict routing to specific DEXes |
| `.blockNumber(n)` | `"latest"`  | Quote against a specific block |
| `.rpcUrls(...)`   | client RPC  | Override RPC endpoints the quoter uses |
| `.get()`          | —           | Fetch and return a `Quote` |
| `.execute(opts?)` | —           | Fetch the quote, then execute. Requires signer. |

#### Go

```go
client.GetQuote(ctx context.Context, opts ...QuoteOption) (*Quote, error)
client.Swap(ctx context.Context, opts ...QuoteOption) (*SwapResult, error)
```

| Option                   | Default      | Description |
|--------------------------|--------------|-------------|
| `From(in, out, amount)`  | **required** | Token pair + input amount |
| `WithSlippage(v)`        | `0.5`        | Slippage in percent |
| `WithMaxHops(n)`         | `2`          | Max route hops |
| `OnNetwork(n)`           | `NetworkBase`| Target network |
| `WithPriceBase(s)`       | —            | Same as `.priceBase` |
| `WithDexs(...)`          | —            | Restrict routing |
| `WithBlockNumber(n)`     | `"latest"`   | Quote against a specific block |
| `WithRpcUrls(...)`       | client RPC   | Override RPC endpoints the quoter uses |

#### Quote

```typescript
interface Quote {
  tokenIn:           Address    // input token
  tokenOut:          Address    // output token
  amountIn:          string     // human-readable input
  amountOut:         string     // human-readable estimated output
  minOut:            string     // human-readable minimum after slippage
  amountInWei:       bigint     // exact input — pass to approve()
  amountOutWei:      bigint     // estimated output
  minOutWei:         bigint     // minimum output enforced on-chain
  steps:             Hex        // encoded route — passed to AFI.swap()
  path:              Address[]  // token addresses in the route
  hops:              Hop[]      // per-hop breakdown
  slippage:          number     // applied slippage in percent
  feeBps:            number     // protocol fee at quote time
  tokenInPrice:      string     // input price in tokenOut units
  tokenOutPrice:     string     // output price in tokenIn units
  tokenInBasePrice?: string     // populated by priceBase()
  tokenOutBasePrice?: string    // populated by priceBase()
  createdAt:         number     // unix-ms timestamp — used by isQuoteStale()
}

interface Hop {
  tokenIn:       Address
  tokenOut:      Address
  amountIn:      string
  amountOut:     string
  minOut:        string
  amountInWei:   bigint
  amountOutWei:  bigint
  minOutWei:     bigint
  tokenInPrice:  string
  tokenOutPrice: string
  slippage:      number
  type:          string    // pool protocol, e.g. "v3", "v2"
  kind:          string    // routing engine
  routeId:       number
  weight:        number
}
```

---

### Write operations (require a signer)

#### `connect(privateKey)` / `Connect(privateKey)`

Attaches a signer. Accepts hex with or without the `0x` prefix.

```typescript
client.connect("0x…")        // returns this
const c = new AfiClient({ rpcUrl, privateKey: "0x…" })
```

```go
err := client.Connect("…")
```

`client.address()` (TS) / `client.Address()` (Go) returns the derived wallet
address, or the zero address when no signer is set.

#### `approve(token, amountWei)` / `Approve(ctx, token, amountWei)`

Submits an exact-amount approval for the AFI router. Returns a `PendingTx`
(with the hash available immediately) or **null** when the existing allowance
is already enough — saving a transaction.

The SDK resets the allowance to zero first for USDT-style tokens. If the reset
itself fails (and the subsequent approve also fails), both errors are surfaced
in the resulting `ApprovalError`.

```typescript
const pending = await client.approve(quote.tokenIn, quote.amountInWei)
if (pending) {
  console.log("Approval tx:", pending.txHash)
  await pending.wait()
}
```

```go
pending, err := client.Approve(ctx, quote.TokenIn, quote.AmountInWei)
if pending != nil {
    receipt, err := pending.Wait(ctx)
}
```

#### `simulate(quote)` / `Simulate(ctx, quote)`

Runs an `eth_call` against the AFI router. Resolves (or returns nil) on success.
Throws `SimulationFailedError` (TS) / returns `*AfiError{Code:"SIMULATION_FAILED"}`
(Go) carrying the revert reason when the swap would revert. **No transaction
is sent in either case.**

```typescript
try {
  await client.simulate(quote)
} catch (e) {
  if (isSimulationFailedError(e)) console.error("would revert:", e.reason)
}
```

```go
if err := client.Simulate(ctx, quote); err != nil {
    log.Println("would revert:", err)
}
```

#### `submitSwap(quote)` / `SubmitSwap(ctx, quote)`

Sends the swap transaction without waiting for confirmation. Returns a
`PendingSwap` whose `wait(opts?)` blocks until confirmed.

#### `executeSwap(quote, opts?)` / `ExecuteSwap(ctx, quote, opts?)`

Runs the full sequence — balance check → approve → simulate → submit → wait.
Returns when the swap has been confirmed.

```typescript
interface ExecuteOptions {
  confirmations?: number    // default: 1
  timeoutMs?:     number    // default: none
}
```

```go
type ExecuteOptions struct {
    Confirmations uint64
    TimeoutMs     int64
}
```

#### `swap(opts)` / `Swap(ctx, opts...)`

Convenience: fetches a quote, then runs `executeSwap` with default options.
Use the staged flow or `executeSwap(quote, opts)` when you need confirmations,
timeouts or user confirmation between quote and execution.

#### `estimateSwapCost(quote)` / `EstimateSwapCost(ctx, quote)`

Projects the gas cost without broadcasting a transaction. Returns
[`SwapCostEstimate`](#swapcostestimate). Useful for displaying "estimated
network fee" before the user signs.

#### Result types

```typescript
interface PendingTx {
  txHash: Hex
  wait(opts?: WaitForTxOptions): Promise<TxReceipt>
}

interface PendingSwap {
  txHash: Hex
  wait(opts?: WaitForTxOptions): Promise<SwapResult>
}

interface SwapResult {
  txHash:      Hex
  blockNumber: bigint
  amountIn:    bigint     // actual amountIn from the SwapExecuted event
  amountOut:   bigint     // actual amountOut from the SwapExecuted event
  tokenIn:     Address
  tokenOut:    Address
  gasUsed:     bigint
}

interface TxReceipt {
  blockNumber: bigint
  gasUsed:     bigint
}

interface WaitForTxOptions {
  confirmations?: number   // default: 1
  timeoutMs?:     number   // default: none
}
```

---

### Transaction utilities

#### `waitForTx(hash, opts?)` / `WaitForTx(ctx, hash, opts?)`

Polls until the given transaction reaches the requested confirmations. Useful
for hashes obtained outside the SDK (persisted from a prior run, queued in a
job, returned by another service).

```typescript
const receipt = await client.waitForTx("0x…", { confirmations: 2, timeoutMs: 30_000 })
```

```go
receipt, err := client.WaitForTx(ctx, "0x…", afi.WaitForTxOptions{
    Confirmations: 2, TimeoutMs: 30_000, PollIntervalMs: 1_000,
})
```

#### `parseSwapResult(receipt)` / `ParseSwapResult(receipt)`

Decode the AFI `SwapExecuted` event from any receipt. Returns `null` / `nil`
when no `SwapExecuted` log is present (the tx wasn't an AFI swap).

```typescript
import { parseSwapResult } from "@afi-run/sdk"

const result = parseSwapResult(receipt) // SwapResult | null
```

```go
result, err := afi.ParseSwapResult(receipt) // nil when no SwapExecuted log present
```

Use this for indexers, replay tools, queued jobs that store a hash and
reconcile later, and end-to-end tests.

---

### Configuration

| Method | Description |
|---|---|
| `setApiUrl(url)` / `SetApiURL(url)` | Override the base URL of the AFI API (default `https://rpc.afi.run`). |
| `setGasBufferPercent(n)` / `SetGasBufferPercent(n)` | Override the gas buffer at runtime. Pass `0` to disable. |
| `setLogger(fn)` / `SetLogger(fn)` | Attach or replace the diagnostic logger. |
| `clearTokensCache(network?)` / `ClearTokensCache(network?)` | Force the next `getTokens()` to refetch. |

---

## Helpers

### Address utilities

```typescript
import {
  isAddress,
  checksumAddress,    // EIP-55
  isZeroAddress,
  equalAddresses,     // case-insensitive
  ZERO_ADDRESS,
} from "@afi-run/sdk"
```

```go
afi.IsAddress(s)          // requires "0x" prefix (matches viem/ethers)
afi.Checksum(s)           // EIP-55 string
afi.IsZeroAddress(s)
afi.EqualAddresses(a, b)  // case-insensitive
afi.ZeroAddress           // common.Address{}
afi.ZeroAddressHex        // "0x00…00"
```

### Slippage calculator

```typescript
import { calculateMinOut, applySlippage } from "@afi-run/sdk"

const minOut = calculateMinOut(quote.amountOutWei, 0.5)  // 0.5% off, floor-rounded
```

```go
minOut := afi.CalculateMinOut(quote.AmountOutWei, 0.5)
```

`slippagePct` is in percent (`0.5` = 0.5%). Negative values clamp to 0. Values
≥ 100 return 0.

### Unit conversion

```typescript
import { parseUnits, formatUnits } from "@afi-run/sdk"

parseUnits("1000", 6)              // 1_000_000_000n
formatUnits(1_000_000_000n, 6)     // "1000"
```

```go
wei, _ := afi.ParseUnits("1000", 6) // *big.Int
str   := afi.FormatUnits(wei, 6)    // "1000"
```

### Explorer URLs

```typescript
client.txUrl(result.txHash)             // https://basescan.org/tx/…
client.addressUrl(addr, NETWORK.BSC)    // https://bscscan.com/address/…

// Standalone:
import { txUrl, addressUrl, NETWORK_EXPLORERS } from "@afi-run/sdk"
txUrl(hash, NETWORK.BASE, "https://my-explorer")   // custom base
```

```go
url, _ := client.TxURL(result.TxHash.Hex())
addr, _ := afi.AddressURL(walletAddr, afi.NetworkArbitrum)
```

Defaults live in `NETWORK_EXPLORERS` / `afi.NetworkExplorers` and can be
overridden at runtime.

### Quote staleness

```typescript
import { isQuoteStale } from "@afi-run/sdk"

if (isQuoteStale(quote, 30)) {   // older than 30 seconds
  quote = await client.quote(...).get()
}
```

```go
if quote.IsStale(30) {
    quote, _ = client.GetQuote(ctx, ...)
}
```

Quotes go stale fast (a few seconds is typical for volatile pairs). Always
re-quote before broadcasting on a slow flow (hardware wallet confirmation,
multi-sig approval, manual review).

### Custom error decoding (`decodeRevertReason`, decoded fields on errors)

The SDK ships with the **AFI router's 9 custom errors** pre-registered (verified
on Basescan), plus OpenZeppelin's `Ownable*` / `ReentrancyGuardReentrantCall`
and Solidity's built-in `Error(string)` / `Panic(uint256)`. Reverts are decoded
automatically; the structured result is attached to the thrown error.

```typescript
try {
  await client.simulate(quote)
} catch (e) {
  if (isSimulationFailedError(e) && e.decoded) {
    // e.decoded = { name: "InsufficientFunds", signature: "InsufficientFunds(uint256)", args: [100n] }
    if (e.decoded.name === "InsufficientFunds") {
      toast.error(`Pool only has ${e.decoded.args[0]} available`)
    }
  }
}
```

```go
err := client.Simulate(ctx, quote)
var afiErr *afi.AfiError
if errors.As(err, &afiErr) && afiErr.Decoded != nil {
    if afiErr.Decoded.Name == "InsufficientFunds" {
        log.Printf("pool only has %s available", afiErr.Decoded.Args[0])
    }
}
```

#### Decoded errors that ship with the SDK

| Error                              | From          |
|------------------------------------|---------------|
| `DifferentAssets(address,address)` | AFI router    |
| `FeeTooHigh(uint16)`               | AFI router    |
| `InsufficientFunds(uint256)`       | AFI router    |
| `InvalidRouteID(uint16)`           | AFI router    |
| `NotOperator()`                    | AFI router    |
| `OwnableInvalidOwner(address)`     | OpenZeppelin  |
| `OwnableUnauthorizedAccount(address)` | OpenZeppelin |
| `ReentrancyGuardReentrantCall()`   | OpenZeppelin  |
| `ZeroAddress()`                    | AFI router    |
| `Error(string)`                    | Solidity      |
| `Panic(uint256)`                   | Solidity      |

#### Register your own contract's errors

```typescript
import { registerCustomErrors, decodeRevertReason } from "@afi-run/sdk"

registerCustomErrors([
  { type: "error", name: "MyContractError", inputs: [
    { name: "code", type: "uint256" },
    { name: "msg",  type: "string" },
  ]},
])

// Decode raw revert data manually
const decoded = decodeRevertReason("0x…")
// or rely on it being attached to thrown errors going forward
```

```go
// Parse any ABI containing error definitions and register globally.
a, _ := abi.JSON(strings.NewReader(`[{"type":"error","name":"MyContractError", ...}]`))
afi.RegisterCustomErrors(a)

decoded := afi.DecodeRevertReason(rawHexBytes) // *afi.DecodedRevert
```

---

### Transaction fee in results

`SwapResult` and `TxReceipt` now expose the effective fee paid:

```typescript
const result = await client.executeSwap(quote)
console.log(`Tx cost: ${result.feeEth} ETH (${result.feeWei} wei @ ${result.effectiveGasPrice} wei/gas)`)
```

Same fields available on the Go side:

```go
fmt.Printf("Tx cost: %s ETH\n", result.FeeETH)
```

Receipts returned by `waitForTx`, `pending.wait()`, `pending.wait()` of
`approve`/`revoke` all carry the fee.

---

### `getTxStatus(hash)` — non-blocking status

Returns immediately with the current state of a tx — useful for UI polling
indicators where blocking on a receipt would be a bad idea.

```typescript
const status = await client.getTxStatus(hash)
// "pending" | "success" | "failed" | "unknown"
```

```go
status, err := client.GetTxStatus(ctx, hash)
switch status {
case afi.TxStatusPending: // …
case afi.TxStatusSuccess: // …
}
```

---

### `getTokenPrice(in, out, opts?)` — light price lookup

Quick exchange-rate check for a token pair without committing to a swap.

```typescript
const { price, inverse } = await client.getTokenPrice(USDC, WETH)
// price   = "0.00031" (1 USDC in WETH)
// inverse = "3225"    (1 WETH in USDC)

// Override defaults:
await client.getTokenPrice(USDC, WETH, { amount: "1000", slippage: 1.0, network: NETWORK.BSC })
```

```go
p, _ := client.GetTokenPrice(ctx, usdc, weth)
// p.Price, p.Inverse
```

---

### Nonce management — `getNonce`, `useManagedNonce`

For bots that submit multiple swaps in parallel without waiting between them.

```typescript
// One-off read
const n = await client.getNonce()

// Managed mode (recommended for bots)
await client.useManagedNonce()    // syncs from chain, then maintains a local counter
await Promise.all([
  client.executeSwap(quote1),
  client.executeSwap(quote2),
  client.executeSwap(quote3),     // each gets a unique nonce, no race
])

// On error / fork / replacement
await client.resetManagedNonce()  // re-syncs the counter

// Per-call override
await client.executeSwap(quote, { nonce: 142 })
```

```go
n, _ := client.GetNonce(ctx)
client.UseManagedNonce(ctx)
nonce := uint64(142)
client.ExecuteSwap(ctx, quote, afi.ExecuteOptions{Nonce: &nonce})
client.ResetManagedNonce(ctx)
client.DisableManagedNonce()
```

When the managed counter drifts (rejected tx, replacement), call
`resetManagedNonce()` / `ResetManagedNonce(ctx)` to re-sync from the chain.

---

### `preflight(quote)` — combined readiness check

Runs balance + allowance + simulate **without sending any tx** and returns a
structured report so your UI can drive a "ready to swap" indicator.

```typescript
const report = await client.preflight(quote)
if (!report.canExecute) {
  for (const p of report.problems) console.error(`${p.code}: ${p.message}`)
} else if (report.needsApproval) {
  showButton("Approve & Swap")
} else {
  showButton("Swap")
}
```

```go
report, _ := client.Preflight(ctx, quote)
if !report.CanExecute {
    for _, p := range report.Problems {
        fmt.Printf("%s: %s\n", p.Code, p.Message)
    }
}
```

`canExecute = problems.length === 0` — `needsApproval` is informational because
`executeSwap` will auto-handle it.

---

### Pre-encoded transactions (`encodeSwap`, `encodeApprove`, `encodeRevoke`)

For frontends where the private key lives in a wallet connector (Wagmi,
RainbowKit, MetaMask, Frame, hardware wallets, Safe SDK), build the calldata
with the SDK and submit it through the connector.

```typescript
import { encodeSwap, encodeApprove } from "@afi-run/sdk"

const approveTx = encodeApprove(quote.tokenIn, quote.amountInWei)
await walletClient.sendTransaction(approveTx)

const swapTx = encodeSwap(quote)
const hash = await walletClient.sendTransaction(swapTx)
```

```go
swapTx, _ := afi.EncodeSwap(quote)         // {To, Data, Value}
approveTx, _ := afi.EncodeApprove(usdc, amt)
revokeTx, _ := afi.EncodeRevoke(usdc)
```

All three are also exposed as client methods (`client.encodeSwap(quote)` etc.)
when you already have a configured client.

---

### Revoke allowance — `revoke(token)` / `Revoke(ctx, token)`

Sends `approve(AFI, 0)` to zero the router's allowance. Returns `null` /
`nil` when the allowance is already zero. Use as a security cleanup after a swap.

```typescript
const tx = await client.revoke(quote.tokenIn)
await tx?.wait()
```

```go
pending, err := client.Revoke(ctx, quote.TokenIn)
if pending != nil {
    pending.Wait(ctx)
}
```

---

### Generic multicall — `multicall(calls)` / `Multicall(ctx, calls)`

Bundle arbitrary read calls into a single RPC round-trip via Multicall3. Use
for any batched read beyond `tokenInfo` — pool prices, your own contracts,
custom DEX state.

```typescript
import { ERC20_ABI } from "@afi-run/sdk"

const results = await client.multicall([
  { address: usdc, abi: ERC20_ABI, functionName: "totalSupply" },
  { address: weth, abi: ERC20_ABI, functionName: "totalSupply" },
])
for (const r of results) {
  if (r.status === "success") console.log(r.result)
}
```

```go
// Multicall3 ABI is exposed as afi.Multicall3ABIJSON for low-level use.
calls := []afi.Multicall3Call{ /* … */ }
results, err := client.Multicall(ctx, calls)
```

---

### Refresh a stale quote — `refreshQuote(quote)` / `RefreshQuote(ctx, quote)`

Re-fetches a quote using its original parameters (network, slippage, maxHops,
priceBase, dexs). Convenience for slow flows (hardware-wallet confirmation,
multi-sig review) where the original builder context is lost.

```typescript
if (isQuoteStale(quote, 30)) {
  quote = await client.refreshQuote(quote)
}
```

```go
if quote.IsStale(30) {
    quote, _ = client.RefreshQuote(ctx, quote)
}
```

---

### Token metadata cache

`tokenInfo` / `tokenInfoBatch` keep `(symbol, name, decimals)` in an
in-memory cache — these never change for an ERC-20 token, so the second
metadata-only lookup costs **zero RPC calls**. When `owner` is provided, only
the live balance/allowance are fetched on subsequent calls.

Wipe the cache with `clearTokenMetadataCache()` (TS) /
`ClearTokenMetadataCache()` (Go) if you change RPC providers and want to
re-verify token shapes.

---

### JSON serialization

`Quote`, `SwapResult` and `TokenInfo` contain `bigint` / `*big.Int` fields,
which break `JSON.stringify` and lose precision in `json.Marshal`. The SDK
provides round-trip helpers (bigints become base-10 strings).

```typescript
import {
  bigintReplacer,
  quoteToJSON,    quoteFromJSON,
  swapResultToJSON, swapResultFromJSON,
  tokenInfoToJSON,  tokenInfoFromJSON,
} from "@afi-run/sdk"

// Save
await db.put(`quote:${id}`, JSON.stringify(quoteToJSON(quote)))
// Generic alternative for arbitrary objects:
JSON.stringify(anyObject, bigintReplacer)

// Restore
const restored = quoteFromJSON(await db.get(`quote:${id}`))
```

```go
// Quote / SwapResult / TokenInfo implement MarshalJSON and UnmarshalJSON natively.
data, _ := json.Marshal(quote)        // bigints serialized as strings
var q afi.Quote
_ = json.Unmarshal(data, &q)
```

### Exported ABIs

```typescript
import { AFI_ABI, ERC20_ABI, MULTICALL3_ABI } from "@afi-run/sdk"
// drop-in for viem readContract / writeContract / parseEventLogs
```

```go
// Raw JSON strings — feed to abi.JSON(strings.NewReader(...)).
afi.AFIABIJSON
afi.ERC20ABIJSON
afi.Multicall3ABIJSON
```

---

## Logging & diagnostics

Attach a logger to capture timing and outcomes for major operations.

```typescript
new AfiClient({
  rpcUrl,
  logger: (e) => console.log(`${e.method} ${e.durationMs}ms ok=${e.ok}`),
})

interface LogEvent {
  kind:        "rpc" | "api"
  method:      string        // "getQuote", "approve", "simulate", "submitSwap", "executeSwap"
  durationMs:  number
  ok:          boolean
  error?:      unknown
}
```

```go
afi.NewClient(afi.Config{
    RPCURL: url,
    Logger: func(e afi.LogEvent) {
        log.Printf("%s %dms ok=%v", e.Method, e.DurationMs, e.OK)
    },
})
```

Replace at runtime with `setLogger(fn)` / `SetLogger(fn)`. Pass `undefined` /
`nil` to disable.

`health()` / `Health(ctx)` probes the RPC (chain ID) and the AFI API in
parallel:

```typescript
const h = await client.health()
if (!h.rpc.ok || !h.api.ok) {
  console.error("not ready:", h)
  process.exit(1)
}
```

---

## Error handling

All thrown errors derive from `AfiError` (TS) / `*AfiError` (Go). The `Code`
identifies the failure class; the `Message` is human-friendly; some codes
attach extra context fields.

### Error code reference

| Code | When | Extra fields |
|---|---|---|
| `NO_SIGNER` | A write method was called without `connect()`. | — |
| `INSUFFICIENT_BALANCE` | The wallet's tokenIn balance is below the required amount. | `token`, `owner`, `symbol`, `decimals`, `balance`, `required` |
| `APPROVAL_FAILED` | `approve()` (or the USDT-style reset) reverted. | — |
| `SIMULATION_FAILED` | `eth_call` of `AFI.swap(...)` reverted before any tx was sent. | `reason`, `revertData?` (TS) |
| `QUOTE_FAILED` | The quoter API returned an error (no route, validation failure, …). | — |
| `SWAP_REVERTED` | The swap transaction reverted on-chain or `estimateGas` failed. | `reason` (TS) |

When `INSUFFICIENT_BALANCE` is raised, the SDK performs **one extra
multicall** to attach `symbol` and `decimals` so the message reads
*"Insufficient USDC for 0xABcd…: have 0.5, need 1"* instead of raw addresses.

### TypeScript — type guards

Prefer the named guards over `instanceof` — they survive class shims and
transpiled output across `esbuild`, `swc`, ESM↔CJS interop, etc.

```typescript
import {
  isAfiError,
  isInsufficientBalanceError,
  isQuoteError,
  isSimulationFailedError,
  isApprovalError,
  isSwapRevertedError,
  isNoSignerError,
} from "@afi-run/sdk"

try {
  await client.executeSwap(quote)
} catch (e) {
  if (isInsufficientBalanceError(e)) {
    showFundingPrompt(e.symbol ?? e.token, e.required - e.balance)
  } else if (isSimulationFailedError(e)) {
    toast.error(`Swap would revert: ${e.reason}`)
  } else if (isQuoteError(e)) {
    toast.error("No route found.")
  } else {
    Sentry.captureException(e)
    throw e
  }
}
```

### Go

```go
result, err := client.ExecuteSwap(ctx, quote)
if err != nil {
    var afiErr *afi.AfiError
    if errors.As(err, &afiErr) {
        switch afiErr.Code {
        case "NO_SIGNER":
            log.Println("connect a signer first")
        case "INSUFFICIENT_BALANCE":
            log.Printf("need %s more %s", new(big.Int).Sub(afiErr.Required, afiErr.Balance), afiErr.Symbol)
        case "SIMULATION_FAILED":
            log.Println("would revert:", afiErr.Message)
        case "QUOTE_FAILED":
            log.Println("no route found")
        case "APPROVAL_FAILED":
            log.Println("approval reverted")
        case "SWAP_REVERTED":
            log.Println("on-chain revert")
        }
        return
    }
    log.Fatal(err) // network / encoding / programming error
}
```

---

## Security model

| Risk                         | Mitigation |
|------------------------------|------------|
| Slippage bypass              | `minOutWei` always comes from the quoter; zero values are rejected client-side. |
| Excessive allowance          | Approvals are always for the exact `amountInWei` — never `MAX_UINT256`. |
| USDT-style approval failure  | The SDK resets allowance to 0 before re-approving when needed; failures of the reset are preserved and surfaced if the subsequent approve fails. |
| Reverting transactions       | `simulate` runs before every `executeSwap` — failures throw without spending gas. |
| Allowance race conditions    | Allowance is re-read on-chain after each approve to confirm the new value is reflected. |
| Gas under-estimation         | `eth_estimateGas` is multiplied by `(1 + gasBufferPercent/100)` (default +15 %). |
| Chain mismatch               | The signer reads chain ID from the RPC (Go) and the SDK exposes `chainId()` for callers to verify (TS). |
| Stale quotes                 | `Quote.createdAt` is set at fetch time. Use `isQuoteStale(quote, maxAge)` before submitting. |
| Native ETH passed by mistake | The router does not accept native ETH; always pass WETH. |

The router contract address and the protocol fee are read live from
on-chain on every quote — the SDK never trusts cached values for these.

---

## Recipes

### Save a quote, restore it later

```typescript
import { quoteToJSON, quoteFromJSON, isQuoteStale } from "@afi-run/sdk"

await redis.set(`quote:${userId}`, JSON.stringify(quoteToJSON(quote)))
// …
const raw = await redis.get(`quote:${userId}`)
const quote = quoteFromJSON(raw!)
if (isQuoteStale(quote, 60)) {
  // refetch
}
```

### Skip approve when allowance is already sufficient

```typescript
if (await client.hasAllowance(quote.tokenIn, quote.amountInWei)) {
  // skip approve
} else {
  const tx = await client.approve(quote.tokenIn, quote.amountInWei)
  await tx?.wait()
}
```

```go
ok, _ := client.HasAllowance(ctx, quote.TokenIn, quote.AmountInWei)
if !ok {
    pending, _ := client.Approve(ctx, quote.TokenIn, quote.AmountInWei)
    if pending != nil {
        _, _ = pending.Wait(ctx)
    }
}
```

### Show estimated network fee before signing

```typescript
const cost = await client.estimateSwapCost(quote)
toast.info(`Estimated network fee: ~${cost.totalEth} ETH`)
```

### Portfolio view — batch token info for N tokens

```typescript
const tokens = await client.getTokens()
const infos = await client.tokenInfoBatch(
  tokens.filter(t => t.active).map(t => t.address),
  "self",
)
infos.forEach(i => console.log(`${i.symbol}: ${formatUnits(i.balance ?? 0n, i.decimals)}`))
```

### Fail fast at startup

```typescript
const h = await client.health()
if (!h.rpc.ok || !h.api.ok) {
  console.error("AFI SDK not ready", h)
  process.exit(1)
}
const net = await client.detectNetwork()
if (net !== "base") {
  console.error(`Expected RPC for base, got ${net}`)
  process.exit(1)
}
```

### Bot that waits for 2 confirmations

```typescript
const result = await client
  .quote(USDC, WETH, "500")
  .slippage(1.0)
  .execute({ confirmations: 2, timeoutMs: 60_000 })
```

```go
result, err := client.ExecuteSwap(ctx, quote, afi.ExecuteOptions{
    Confirmations: 2, TimeoutMs: 60_000,
})
```

### Replay or index a known transaction

```typescript
import { parseSwapResult } from "@afi-run/sdk"
const receipt = await publicClient.getTransactionReceipt({ hash })
const result = parseSwapResult(receipt)
if (result) await indexSwap(result)
```

---

## Networks & constants

### Supported networks

| Network     | Chain ID | Default explorer            | TS constant            | Go constant           |
|-------------|----------|------------------------------|------------------------|------------------------|
| Base        | 8453     | https://basescan.org         | `NETWORK.BASE`         | `afi.NetworkBase`      |
| BSC         | 56       | https://bscscan.com          | `NETWORK.BSC`          | `afi.NetworkBSC`       |
| Arbitrum    | 42161    | https://arbiscan.io          | `NETWORK.ARBITRUM`     | `afi.NetworkArbitrum`  |
| Ethereum    | 1        | https://etherscan.io         | `NETWORK.ETHEREUM`     | `afi.NetworkEthereum`  |
| Unichain    | 130      | https://uniscan.xyz          | `NETWORK.UNICHAIN`     | `afi.NetworkUnichain`  |

Override explorers at runtime via `NETWORK_EXPLORERS` (TS) or
`afi.NetworkExplorers` (Go).

### Supported DEXes

```typescript
import { DEX } from "@afi-run/sdk"
// DEX.UNI_V3 · DEX.UNI_V4 · DEX.CAKE_V3 · DEX.AERODROME
// DEX.BALANCER · DEX.CURVE128 · DEX.CURVE256 · DEX.FLUID
```

```go
// afi.DexUniV3 · afi.DexUniV4 · afi.DexCakeV3 · afi.DexAerodrome
// afi.DexBalancer · afi.DexCurve128 · afi.DexCurve256 · afi.DexFluid
```

### Deployed contracts per chain

Deployed 2026-05-30. All contracts verified on the respective block explorer.

**Afi (user swap router)** — `AFI_ADDRESSES` (TS) / `afi.AfiAddresses` (Go):

| Chain | Address |
|---|---|
| Ethereum (1) | `0xc578a4e89795803F396160610F4990c44abA8dAb` |
| BSC (56) | `0xFd4F8822f13D01aB142Bc985Ce587E35d7673C6e` |
| Unichain (130) | `0xFd4F8822f13D01aB142Bc985Ce587E35d7673C6e` |
| Base (8453) | `0xFd4F8822f13D01aB142Bc985Ce587E35d7673C6e` |
| Arbitrum (42161) | `0xd74F60BD38243d089e286E3B6b9348f43a2314dF` |

**RouteQuoter (off-chain simulation via `eth_call` + `state_override`)** — `ROUTE_QUOTER_ADDRESSES` (TS) / `afi.RouteQuoterAddresses` (Go):

| Chain | Address |
|---|---|
| Ethereum | `0x5e41b417E9742DB9c5402F8B1969a33891628Bed` |
| BSC | `0xcA37E05a20E93fD88E5367F9d7d1422937c57A38` |
| Unichain | `0x2Cc852Cd57CC1b57CA09dbA7f69F0e225008cEBE` |
| Base | `0xB5637138Cee6e757B679FFF8aDEA8DBa3E7544bB` |
| Arbitrum | `0xBdD42B4fF06aCa8908D5E5d4826fFf5cdaC43895` |

**AfiReferralRouter (referral-fee wrapper)** — `REFERRAL_ROUTER_ADDRESSES` (TS) / `afi.ReferralRouterAddresses` (Go). Deployed 2026-06-03:

| Chain | Address |
|---|---|
| Ethereum (1) | `0x47E7cE4237130F02202e081Efa1Fd338F23Ead77` |
| BSC (56) | `0x7356960324a627994bb5959CF615DC5f2B38B738` |
| Unichain (130) | `0xcdC506dEA82FE7d034C0281564d0dbe49171D242` |
| Base (8453) | `0x2dC7a3990618baa91c450521004F14A334BF47c6` |
| Arbitrum (42161) | `0x9DaD9322e196F734Fa25eC3b0db90387945B397C` |

### Other constants

| Name                | Value |
|---------------------|-------|
| Multicall3 (all)    | `0xcA11bde05977b3631167028862bE2a173976CA11` |
| Permit2 (all)       | `0x000000000022D473030F116dDEE9F6B43aC78BA3` |
| API base URL        | `https://rpc.afi.run` |
| Default gas buffer  | `15` (%) |

```typescript
import {
  AFI_ADDRESSES,           // Record<chainId, Address>
  ROUTE_QUOTER_ADDRESSES,  // Record<chainId, Address>
  AFI_ABI,
  MULTICALL3_ADDRESS,
  DEFAULT_GAS_BUFFER_PERCENT,
  NETWORK_CHAIN_IDS,
} from "@afi-run/sdk"
```

```go
afi.AfiAddresses           // map[Network]common.Address
afi.RouteQuoterAddresses   // map[Network]common.Address
afi.AfiABI                 // string (JSON)
afi.Multicall3Address      // common.Address
afi.DefaultGasBufferPercent
afi.NetworkChainIDs        // map[Network]int64
```

### Low-level helpers

For users building operator or arbitrage flows directly (not via the high-level
`client.swap()` path).

**Tight-format step encoder** — produces the bytes consumed by `Lib.runRoutes`:

```typescript
import { encodeSteps, type Step } from "@afi-run/sdk"

const steps: Step[] = [
  { id: 3, data: "0x...59-byte-stepData..." }, // UniV3 route id=3
]
const params = encodeSteps(steps) // -> Hex
// Pass `params` as the last arg of Afi.swap / Afi.swapFor
```

```go
import afi "github.com/afi-run/sdk/go"

params, err := afi.EncodeSteps([]afi.Step{
    {ID: 3, Data: stepData}, // UniV3
})
```

Layout: `uint8 numSteps + [uint16 id | uint16 dataLen | bytes data] × N`.

**When to use `swap` / `swapFor` / `batchSwapFor`**

The Afi router exposes three execution entrypoints. Pick the one that matches who pays the gas and where the input tokens come from:

| Function | Caller | Use case |
|---|---|---|
| `Afi.swap(tokenIn, amount, tokenOut, minOut, params)` | end-user (msg.sender pays) | DApp user swapping their own tokens |
| `Afi.swapFor(user, tokenIn, amount, tokenOut, minOut, params)` | operator (operator pulls from `user`) | Bot executing on behalf of a pre-approved user |
| `Afi.batchSwapFor(SwapRequest[])` | operator | Many users in one tx — gas-efficient batch |

Important: `swapFor` and `batchSwapFor` require `user` to have called `IERC20(tokenIn).approve(Afi, amount)` first. Without that allowance, the operator's `transferFrom` will revert.

**Admin / owner encoders**

The SDK now exposes Afi owner-only encoders for dashboards and governance flows. Calldata builders only — broadcasting is the caller's responsibility:

```typescript
import {
  encodeAfiPause, encodeAfiUnpause,
  encodeAfiSetTreasury, encodeAfiSetFeeBps,
  encodeAfiSetUserFeeBps, encodeAfiSetUserFeeBpsBatch,
  encodeAfiClearUserFeeBps, encodeAfiResetAnyUserOverride,
  encodeAfiAddRule, encodeAfiClearRules,
  encodeAfiSetOperator, encodeAfiRescueTokens,
} from "@afi-run/sdk"
```

```go
// afi.EncodeAfiPause / EncodeAfiUnpause / EncodeAfiSetTreasury / EncodeAfiSetFeeBps /
// EncodeAfiSetUserFeeBps / EncodeAfiSetUserFeeBpsBatch / EncodeAfiClearUserFeeBps /
// EncodeAfiResetAnyUserOverride / EncodeAfiAddRule / EncodeAfiClearRules /
// EncodeAfiSetOperator / EncodeAfiRescueTokens
```

The caller of these txs must be the Afi owner. Fee changes are bounded on-chain by `MAX_FEE_BPS = 50` (0.50%) — any value above that reverts with `FeeTooHigh`.

**Event parsers**

For indexers, dashboards, and post-tx UI updates, the SDK ships typed parsers for every event emitted by Afi. Each helper takes the receipt's `logs` and returns an array of decoded events (empty when no matching log is present):

```typescript
import {
  parseSwapExecuted, parseFeeCollected, parseTreasuryUpdated,
  parseFeeBpsUpdated, parseUserFeeBpsSet, parseUserFeeBpsCleared,
} from "@afi-run/sdk"

// Usage:
const events = parseSwapExecuted(receipt.logs)
// events: [{ from, assetIn, amountIn, assetOut, amountOut }, ...]
```

```go
// Go SDK equivalents: afi.ParseSwapExecuted, afi.ParseFeeCollected,
// afi.ParseTreasuryUpdated, afi.ParseFeeBpsUpdated, afi.ParseUserFeeBpsSet,
// afi.ParseUserFeeBpsCleared
```

Use these to drive ledgers (`FeeCollected`), governance dashboards (`TreasuryUpdated`, `FeeBpsUpdated`) and per-user fee overrides (`UserFeeBpsSet` / `UserFeeBpsCleared`).

---

## Operator workflows

End-to-end snippets for the operator surfaces of the protocol. They all
share the same pattern: build the route via the quoter (or `encodeSteps`),
sign with the operator key, broadcast.

### Swap on behalf of one user — `swapFor`

`Afi.swapFor` lets an operator execute a quote where the input tokens come from
`user` (who must have `approve(Afi, amount)` first) and the output goes back to
the same `user`. The operator pays gas — the user pays no native ETH.

```typescript
import { AfiClient, NETWORK } from "@afi-run/sdk"

const client = new AfiClient({ rpcUrl: process.env.RPC_URL!, privateKey: process.env.OP_KEY! as `0x${string}` })

const result = await client.swapFor({
  user:     "0xUser...",
  tokenIn:  USDC,
  tokenOut: WETH,
  amountIn: "1000",
  slippage: 0.5,
  network:  NETWORK.BASE,
})
console.log("Swapped for", result.user, "tx:", client.txUrl(result.txHash))
```

### Batch — `batchSwapFor`

Execute up to ~N quotes in a single transaction. Gas-optimal when you have
multiple pre-approved users to settle in the same block.

```typescript
const results = await client.batchSwapFor([
  { user: "0xUserA...", tokenIn: USDC, tokenOut: WETH, amountIn: "500" },
  { user: "0xUserB...", tokenIn: USDC, tokenOut: WETH, amountIn: "750" },
  { user: "0xUserC...", tokenIn: DAI,  tokenOut: WETH, amountIn: "1000" },
], { slippage: 0.5 })

for (const r of results) console.log(r.user, "->", r.amountOut)
```

---

## Admin / governance

The Afi router is `Ownable` — these flows require the **owner key** (not just an
operator). Read live state first, then craft and submit the tx.

### Inspect current state

```typescript
const paused = await client.isPaused()
const feeBps = await client.getFeeBps()
const userFee = await client.getUserFeeBps("0xUser...")  // 0 if no override
console.log({ paused, globalFeeBps: feeBps, userOverrideBps: userFee })
```

### Pause / unpause

```typescript
const tx = await client.adminPause()        // halts new swaps; existing pending tx still finish
await tx.wait()
// ...
await (await client.adminUnpause()).wait()
```

### Change protocol fee

Bounded on-chain by `MAX_FEE_BPS = 50` (0.50%). Values above revert with `FeeTooHigh`.

```typescript
await (await client.adminSetFeeBps(25)).wait()                 // 0.25% global fee
await (await client.adminSetUserFeeBps("0xVIP...", 5)).wait()  // 0.05% for a single user
await (await client.adminClearUserFeeBps("0xVIP...")).wait()   // remove the override
```

### Add a validation rule

Rules are external contracts that implement `IAfiRule`. The router calls each
registered rule before every swap; any revert aborts the swap.

```typescript
await (await client.adminAddRule("0xRule...")).wait()
await (await client.adminClearRules()).wait()
```

The signing key for all `admin*` calls must equal the Afi owner address —
otherwise `OwnableUnauthorizedAccount` is raised. The same applies to
`adminSetTreasury`, `adminSetOperator(addr, true|false)` and
`adminRescueTokens(token, amount, to)`.

---

## Referral router

`AfiReferralRouter` is a thin wrapper in front of `Afi`: it forwards the swap and
may charge a referral fee of up to **0.10%** (`REFERRAL_HARD_CAP_BPS = 10`) on the
**output** token, credited to a referrer and later withdrawn via `claim`. The
router does not need to be an Afi operator — it uses Afi's public `swap`.

The SDK ships **calldata encoders** for it (no high-level `client.*` wrappers
yet); sign and send the returned calldata with viem (TS) or go-ethereum (Go).

### Swap with a referral fee

```typescript
import {
  encodeSwapWithReferral,
  referralRouterAddress,
  REFERRAL_HARD_CAP_BPS,
} from "@afi-run/sdk"

const data = encodeSwapWithReferral({
  tokenIn, amountIn: 1_000_000n, tokenOut,
  minOut: 990_000n,            // enforced AFTER the referral fee
  params,                       // Afi route params (see step builders)
  referrer,                     // pass the zero address to disable the fee
  referralBps: REFERRAL_HARD_CAP_BPS, // <= 10
})
const hash = await wallet.sendTransaction({ to: referralRouterAddress(8453), data })
```

```go
data, _ := afi.EncodeSwapWithReferral(tokenIn, amountIn, tokenOut, minOut, params, referrer, afi.ReferralHardCapBps)
to, _, _ := afi.ReferralRouterAddress(8453)
// sign & send `data` to `to` with go-ethereum
```

`minOut` is enforced by the router on the **net** amount (the router calls Afi
with `minOut = 0` and applies its fee afterward). A referrer claims accrued fees
with `encodeReferralClaim(token, to)` / `encodeReferralClaimMany(tokens, to)`.

### Delegated swaps

A funds owner (A) can authorize a delegate (B) to swap A's tokens, capped by an
amount and a deadline; the **output always returns to A**:

```typescript
// owner A authorizes delegate B for `tokenIn`
encodeSetDelegateAllowance(tokenIn, delegate, 1_000_000n, deadlineUnixSeconds)
// later, B triggers the swap on A's behalf (output goes to A)
encodeSwapWithReferralFor({ user: A, tokenIn, amountIn, tokenOut, minOut, params, referrer, referralBps })
// A can revoke at any time
encodeRevokeDelegate(tokenIn, delegate)
```

There is **no on-chain price protection** on delegated swaps — A trusts B to pass
a fair `minOut` and bounds exposure via the allowance, the deadline, and an ERC20
approval to the router. The effective spendable amount is the MIN of the ERC20
approval and the delegate allowance.

### Owner-only controls

`encodeReferralPause()` / `encodeReferralUnpause()`,
`encodeReferralSetMaxReferralBps(bps)` (`bps <= 10`), and
`encodeReferralRescueTokens(token, to)` (sweeps only the balance **not** owed to
referrers). The signing key must equal the router owner.

See `examples/nodejs/12-referral-swap.ts` and `examples/go/12_referral_swap.go`
for runnable end-to-end flows.

---

## Event indexing

Every event emitted by Afi has a typed parser that takes a `logs`
array and returns decoded entries. Combine with `getLogs` to build indexers,
dashboards, and post-tx ledgers.

```typescript
import {
  parseSwapExecuted, parseFeeCollected,
  parseTreasuryUpdated, parseFeeBpsUpdated,
  parseUserFeeBpsSet, parseUserFeeBpsCleared,
  AFI_ADDRESSES,
} from "@afi-run/sdk"
import { createPublicClient, http, parseAbiItem } from "viem"
import { base } from "viem/chains"

const publicClient = createPublicClient({ chain: base, transport: http(process.env.RPC_URL!) })

// 1. Pull raw logs for the desired range
const logs = await publicClient.getLogs({
  address: [AFI_ADDRESSES[8453]],
  fromBlock: 22_000_000n,
  toBlock:   22_005_000n,
})

// 2. Decode per event type
const swaps         = parseSwapExecuted(logs)         // [{ from, assetIn, amountIn, assetOut, amountOut }]
const fees          = parseFeeCollected(logs)         // protocol fee revenue
const treasury      = parseTreasuryUpdated(logs)

for (const s of swaps) console.log(`${s.from} ${s.amountIn} ${s.assetIn} -> ${s.amountOut} ${s.assetOut}`)
```

Each parser returns `[]` when no matching log is present, so you can chain
them safely. Bigints come through as native `bigint`. The parsers cover:

| Parser | Origin | Use |
|---|---|---|
| `parseSwapExecuted` | Afi | Per-swap settlement |
| `parseFeeCollected` | Afi | Protocol fee ledger |
| `parseTreasuryUpdated` | Afi | Governance audit |
| `parseFeeBpsUpdated` | Afi | Global fee change |
| `parseUserFeeBpsSet` | Afi | Per-user override added |
| `parseUserFeeBpsCleared` | Afi | Per-user override removed |

---

## Per-DEX step builders

For operators who want to skip the HTTP quoter and assemble their own routes
(custom MEV strategies, fully on-chain backtests, integration tests), the SDK
exposes a builder per supported DEX. Each returns the 59-byte `stepData` plus
the route ID expected by `Lib.runRoutes`.

```typescript
import {
  buildUniV3Step, buildCakeV3Step, buildUniV4Step, buildAerodromeStep,
  buildBalancerV3Step, buildFluidStep, buildCurve128Step, buildCurve256Step,
  buildAaveLiquidatorStep,
  encodeSteps,
} from "@afi-run/sdk"
```

| Builder | Required fields |
|---|---|
| `buildUniV3Step({ tokenOut, fee, minOut, sqrtPriceLimitX96 })` | Uniswap V3 pools (fee tier `500/3000/10000`) |
| `buildCakeV3Step({ tokenOut, fee, minOut, sqrtPriceLimitX96 })` | PancakeSwap V3 (same shape as Uni V3) |
| `buildUniV4Step({ currency0, currency1, fee, tickSpacing, hooks, zeroForOne, minOut })` | Uniswap V4 PoolKey + direction |
| `buildAerodromeStep({ pool, tokenOut, tickSpacing, minOut })` | Aerodrome Slipstream — note `tickSpacing` is **int24 signed** |
| `buildBalancerV3Step({ pool, tokenOut, minOut })` | Balancer V3 (one-hop, exactIn) |
| `buildFluidStep({ pool, swap0to1, tokenOut, minOut })` | Fluid DEX pools |
| `buildCurve128Step({ i, j, minDy, pool, tokenOut })` | Curve plain pools with int128 indices |
| `buildCurve256Step({ i, j, minDy, pool, tokenOut })` | Curve cryptoswap / meta with uint256 indices |
| `buildAaveLiquidatorStep({ pool, user, collateralAsset })` | Aave V3 liquidation (special-purpose route) |

### Example — multi-hop route fed directly into `client.swap()`

```typescript
const stepA = buildUniV3Step({
  tokenOut: WETH,
  fee: 500,
  minOut: 0n,                 // intermediate hop — no floor
  sqrtPriceLimitX96: 0n,
})

const stepB = buildCurve128Step({
  i: 2, j: 0,
  minDy: minOutFinalWei,      // floor on the final hop
  pool: "0xCurvePool...",
  tokenOut: USDC,
})

const params = encodeSteps([
  { id: 3, data: stepA.data },   // 3 = UniV3
  { id: 6, data: stepB.data },   // 6 = Curve128
])

const tx = await client.swap({
  tokenIn:    USDC,
  tokenOut:   USDC,             // cycle
  amountInWei,
  minOutWei:  minOutFinalWei,
  params,
})
```

Use this surface when the HTTP quoter is unavailable, when you want
deterministic routes for replay tests, or when your strategy requires pool
parameters the quoter does not expose.

---

## HTTP quoter endpoints

The afi-rpc service ships several endpoints beyond `/quoter`. The client wraps
each one with typed inputs and outputs so you can drop them into TS/Go code
without writing fetch boilerplate.

| Method | Endpoint | Returns | Purpose |
|---|---|---|---|
| `client.findArbitrage(req)` | `POST /arbitrage` | `RouteQuote[]` | Candidate routes for a cycle (set `tokenIn === tokenOut`) |
| `client.findPath(req)` | `POST /command {action:"path"}` | `PathQuote` | Priced multi-hop route for an explicit path |
| `client.getRoutes(req)` | `POST /command {action:"routes"}` | `Route[]` | Candidate token paths for a pair |
| `client.priceQuote(req)` | `POST /command {action:"price"}` | `RouteQuote[]` | Per-DEX quotes for a pair |
| `client.quoteDex(dex, req)` | `POST /command {action:<dex>}` | `RouteQuote[]` | Quotes from a single DEX |
| `client.getLiquidationCandidates(req)` | `POST /aave` | `AavePosition[]` | Eligible Aave V3 positions to liquidate |
| `client.liquidate(req)` | `POST /liquidation-call` | `LiquidationResult` | Repay+swap route for a liquidationCall |

`findArbitrage`, `priceQuote`, and `quoteDex` all return `RouteQuote[]` — a list
of executable single-DEX routes. Pick the best (`routeProfit(r)` =
`amountOutRaw − amountInRaw`) and hydrate an executable `Quote` with
`quoteFromRoute`, which wraps the route's `{routeId, stepData}` hop into the
`Afi.swap` params via `encodeSteps`:

```typescript
import { quoteFromRoute, routeProfit } from "@afi-run/sdk"

// Self-funded cycle: tokenIn === tokenOut, no operator needed.
const routes = await client.findArbitrage({ network: "base", tokenIn: USDC, tokenOut: USDC, amountIn: "1000" })
const best = routes.reduce((a, b) => (routeProfit(b)! > routeProfit(a)! ? b : a))

// Floor output at principal so an unprofitable cycle reverts on-chain.
const quote = quoteFromRoute(best, BigInt(best.amountInRaw))
await client.executeSwap(quote)

const candidates = await client.getLiquidationCandidates({ network: "base" })
if (candidates.length > 0) {
  console.log(candidates[0].user, candidates[0].debtAmount, candidates[0].collaterals)
}
```

> Go mirrors this exactly: `client.FindArbitrage(ctx, req) → []RouteQuote`,
> `afi.QuoteFromRoute(route, minOut)`, `client.ExecuteSwap(ctx, quote)`.

All requests honour the same `rpcUrls` override (per-call) and surface
`QuoteError` on validation failure — wrap them in `isQuoteError(e)` for clean
UX messages.

---

## Migration guide

The SDK exposed a single `AFI_ADDRESS` constant for Base. With the multi-chain
rollout it now ships `AFI_ADDRESSES` (a `Record<chainId, Address>`); the old
constant is removed.

```typescript
// Before
import { AFI_ADDRESS } from "@afi-run/sdk"
const router = AFI_ADDRESS

// After
import { AFI_ADDRESSES } from "@afi-run/sdk"
const router = AFI_ADDRESSES[8453]              // Base
const arbRouter = AFI_ADDRESSES[42161]          // Arbitrum
```

The same shape applies to `ROUTE_QUOTER_ADDRESSES`. Look up
by chain ID — `client.chainId()` returns the value to key with at runtime.

Other surfaces that landed alongside multi-chain:

- **Admin encoders** — `encodeAfiPause`, `encodeAfiSetFeeBps`, `encodeAfiAddRule`, … in `afi-admin.ts`. Use them from a wallet connector when the owner key lives in a hardware/multisig wallet.
- **Event parsers** — typed parsers in `events.ts`; see [Event indexing](#event-indexing).
- **Step builders** — per-DEX `buildXxxStep(...)` helpers; see [Per-DEX step builders](#per-dex-step-builders).
- **HTTP quoter endpoints** — `findArbitrage`, `findPath`, `getRoutes`, `getLiquidationCandidates`, `liquidate`, `priceQuote`, `quoteDex`.

No existing methods changed signatures — `client.swap()`, `client.quote()`,
`client.executeSwap()`, all helpers and all error types remain identical.

---

## Examples directory

| File / directory                       | What it demonstrates |
|----------------------------------------|----------------------|
| `examples/nodejs/1-list-tokens.ts`     | List active tokens on Base and BSC |
| `examples/nodejs/2-get-quote.ts`       | Builder with all options (priceBase, dexs, Token objects) |
| `examples/nodejs/3-execute-swap.ts`    | Quote → review → execute (recommended user flow) |
| `examples/nodejs/4-full-flow.ts`       | One-call `.execute()` |
| `examples/nodejs/5-approve-only.ts`    | Staged: tokenInfo → hasAllowance → approve → simulate → submit → wait |
| `examples/nodejs/6-operator-batch.ts`  | `swapFor` + `batchSwapFor` for pre-approved users |
| `examples/nodejs/9-admin-governance.ts`| Pause, fee bps, rules — owner-only flows |
| `examples/nodejs/10-event-indexer.ts`  | `getLogs` + Afi event parsers |
| `examples/nodejs/12-referral-swap.ts`  | Referral fee, delegation, owner caps (AfiReferralRouter) |
| `examples/go/list-tokens/`             | List active tokens |
| `examples/go/get-quote/`               | Functional options |
| `examples/go/execute-swap/`            | Quote → review → execute |
| `examples/go/full-flow/`               | One-call `Swap()` |
| `examples/go/approve-only/`            | Staged flow |
| `examples/go/operator-batch/`          | `SwapFor` + `BatchSwapFor` |
| `examples/go/admin-governance/`        | Owner-only flows |
| `examples/go/event-indexer/`           | Event parsers |
| `examples/go/referral-swap/`           | Referral fee, delegation, owner caps (AfiReferralRouter) |

Run TS examples:

```bash
cd nodejs && npm install
npx ts-node ../examples/nodejs/1-list-tokens.ts
```

Run Go examples:

```bash
cd examples/go && go mod tidy
go run ./list-tokens
```

Or build every example into `bin/` (git-ignored) via the Makefile:

```bash
cd examples/go
make build              # compiles each example into bin/<name>
make run EX=get-quote   # or run one directly
make list               # show discovered examples
```

---

## Development

### Build from source

```bash
# TypeScript
cd nodejs
npm install
npm run build       # writes to dist/
npm run typecheck   # tsc --noEmit
npm test            # vitest

# Go
cd go
go mod tidy
go build ./...
go test ./...
```

### Project layout

```
afi-sdk/
├── nodejs/          ── @afi-run/sdk (TypeScript)
│   ├── src/
│   │   ├── client.ts, builder.ts          ── public client + quote builder
│   │   ├── token.ts, multicall.ts         ── ERC-20 reads + Multicall3
│   │   ├── swap.ts, quoter.ts             ── swap + quote pipelines
│   │   ├── address.ts, slippage.ts        ── DX helpers
│   │   ├── serialize.ts, explorer.ts      ── JSON + URL helpers
│   │   ├── errors.ts, types.ts            ── error classes + public types
│   │   ├── constants.ts, utils.ts         ── ABIs, addresses, units
│   │   └── index.ts                       ── public exports
│   └── src/__tests__/                     ── 159 unit tests
├── go/              ── github.com/afi-run/sdk/go
│   ├── client.go, options.go              ── public client + functional options
│   ├── token.go, multicall.go             ── ERC-20 reads + Multicall3
│   ├── swap.go, quoter.go                 ── swap + quote pipelines
│   ├── address.go, slippage.go            ── DX helpers
│   ├── serialize.go, explorer.go          ── JSON + URL helpers
│   ├── errors.go, types.go                ── error type + public types
│   └── *_test.go                          ── 159 unit tests
└── examples/        ── runnable end-to-end examples
```

### Git hooks — `pre-push` mirrors CI locally

The repo ships a pre-push hook that runs the same gates as CI before any
`git push`, so you find problems in seconds instead of waiting for a green PR.

```bash
# One-time install (sets core.hooksPath = scripts/git-hooks)
bash scripts/install-hooks.sh
```

On `git push`, the hook detects which subprojects changed and runs:

| Subproject | Steps |
|------------|-------|
| **Node.js** (when `nodejs/` changed) | `typecheck` · `vitest run --coverage` (≥95% stmts/lines/fns, ≥90% branches) · `npm audit --audit-level=high` |
| **Go** (when `go/` or `examples/go/` changed) | `go vet` · `make test-coverage` (≥95%) · `go build` examples · `govulncheck` (auto-installs if missing) |

Knobs:

```bash
SKIP_PRE_PUSH=1 git push      # bypass once
PRE_PUSH_ALL=1  git push      # ignore changed-path detection, run everything
git config --unset core.hooksPath   # remove the hook entirely
```

### Testing strategy

The SDK is **fully unit-tested** with no external dependencies: RPC and HTTP
calls are mocked. Run the full test suite with `npm test` or `go test ./...`.

When testing your own code, mock the `AfiClient` / `*afi.Client` boundary —
the SDK already trusts that the RPC is sane, and that's the cleanest place
for your tests to inject fixtures.

---

## License

MIT © AFI Run contributors. See the [LICENSE](./LICENSE) file for details.
