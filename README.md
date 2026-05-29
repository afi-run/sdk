# AFI SDK

SDK for token swaps via the [AFI Protocol](https://afi.run).

Available in **Node.js (TypeScript)** and **Go**. Supports Base, BSC, Arbitrum, Ethereum, and Unichain.

---

## Install

```bash
# Go
go get github.com/afi-run/sdk/go

# Node.js
git clone https://github.com/afi-run/sdk.git
npm install ./sdk/nodejs
```

> Once the package is published on npm: `npm install @afi-run/sdk`

---

## Quick start

### TypeScript

```typescript
import { AfiClient, NETWORK, DEX, formatUnits } from "@afi-run/sdk"

const client = new AfiClient({ rpcUrl: "https://rpc.ankr.com/base/YOUR_API_KEY" })

// Fetch a quote
const quote = await client
  .quote("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913", // USDC
         "0x4200000000000000000000000000000000000006", // WETH
         "1000")
  .slippage(0.5)
  .get()

console.log(`You get: ~${quote.amountOut} WETH`)
console.log(`Minimum: ${quote.minOut} WETH`)

// Connect signer and execute
client.connect("0xYOUR_PRIVATE_KEY")
const result = await client.executeSwap(quote)
console.log("Tx:", result.txHash)
console.log("Got:", formatUnits(result.amountOut, 18), "WETH")

// Or in one call
const result2 = await client
  .quote(USDC, WETH, "500")
  .slippage(1.0)
  .execute()
```

### Go

```go
client, _ := afi.NewClient(afi.Config{RPCURL: "https://rpc.ankr.com/base/YOUR_API_KEY"})
defer client.Close()

// Fetch a quote
quote, _ := client.GetQuote(ctx,
    afi.From(common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"), afi.WETH, "1000"),
    afi.WithSlippage(0.5),
)
fmt.Printf("You get: ~%s WETH\n", quote.AmountOut)

// Connect signer and execute
client.Connect("YOUR_PRIVATE_KEY")
result, _ := client.ExecuteSwap(ctx, quote)
fmt.Println("Got:", afi.FormatUnits(result.AmountOut, 18), "WETH")

// Or in one call
result2, _ := client.Swap(ctx,
    afi.From(usdc, afi.WETH, "500"),
    afi.WithSlippage(1.0),
)
```

---

## API reference

### `getTokens(network?)` / `GetTokens(ctx, network?)`

```typescript
const tokens = await client.getTokens()               // Base (default)
const bscTokens = await client.getTokens(NETWORK.BSC) // BSC

// Token shape: { address, symbol, decimals, active }
```

```go
tokens, _ := client.GetTokens(ctx)                    // Base (default)
bscTokens, _ := client.GetTokens(ctx, afi.NetworkBSC) // BSC
```

---

### `quote(tokenIn, tokenOut, amountIn)` / `GetQuote(ctx, ...opts)`

Returns a `QuoteBuilder` (TypeScript) or uses functional options (Go).

**Builder methods (TypeScript):**

| Method | Default | Description |
|---|---|---|
| `.slippage(v)` | `0.5` | Slippage tolerance percentage |
| `.maxHops(n)` | `2` | Maximum number of route hops |
| `.network(n)` | `NETWORK.BASE` | Target network |
| `.priceBase(s)` | — | Base asset for price fields; populates `tokenInBasePrice`/`tokenOutBasePrice` |
| `.dexs(...dexs)` | — | Restrict routing to specific DEXes |
| `.blockNumber(n)` | `"latest"` | Quote at a specific block |
| `.rpcUrls(...urls)` | client default | Custom RPC endpoints |
| `.get()` | — | Fetch and return `Quote` |
| `.execute()` | — | Fetch and execute swap (requires signer) |

```typescript
// Accepts address strings or Token objects
const tokens = await client.getTokens()
const usdc = tokens.find(t => t.symbol === "USDC")!
const weth = tokens.find(t => t.symbol === "WETH")!

const quote = await client
  .quote(usdc, weth, "1000")         // Token objects accepted directly
  .slippage(0.5)
  .maxHops(3)
  .network(NETWORK.BASE)
  .priceBase("USDC")                 // optional — adds tokenInBasePrice/tokenOutBasePrice
  .dexs(DEX.UNI_V3, DEX.AERODROME)  // optional — restrict DEXes
  .get()

if (quote.tokenInBasePrice) {
  console.log(`USDC base price: $${quote.tokenInBasePrice}`)
}
```

**Functional options (Go):**

| Option | Default | Description |
|---|---|---|
| `From(tokenIn, tokenOut, amountIn)` | required | Token pair and input amount |
| `WithSlippage(v)` | `0.5` | Slippage tolerance percentage |
| `WithMaxHops(n)` | `2` | Maximum number of route hops |
| `OnNetwork(n)` | `NetworkBase` | Target network |
| `WithPriceBase(s)` | — | Base asset for price fields |
| `WithDexs(dexs...)` | — | Restrict routing to specific DEXes |
| `WithBlockNumber(n)` | `"latest"` | Quote at a specific block |
| `WithRpcUrls(urls...)` | client default | Custom RPC endpoints |

```go
// From() accepts common.Address or afi.Token
tokens, _ := client.GetTokens(ctx)
var usdc, weth afi.Token
for _, t := range tokens {
    switch t.Symbol {
    case "USDC": usdc = t
    case "WETH": weth = t
    }
}

quote, _ := client.GetQuote(ctx,
    afi.From(usdc, weth, "1000"),             // Token objects accepted directly
    afi.WithSlippage(0.5),
    afi.WithMaxHops(3),
    afi.OnNetwork(afi.NetworkBase),
    afi.WithPriceBase("USDC"),                // optional
    afi.WithDexs(afi.DexUniV3, afi.DexAerodrome), // optional
)
if quote.TokenInBasePrice != "" {
    fmt.Printf("USDC base price: $%s\n", quote.TokenInBasePrice)
}
```

**`Quote` fields:**

| Field | Type | Description |
|---|---|---|
| `tokenIn` | `Address` | Input token address |
| `tokenOut` | `Address` | Output token address |
| `amountIn` | `string` | Human-readable input amount |
| `amountOut` | `string` | Human-readable estimated output |
| `minOut` | `string` | Minimum output after slippage |
| `amountInWei` | `bigint` | Input as Wei — use for `approve()` |
| `amountOutWei` | `bigint` | Estimated output as Wei |
| `minOutWei` | `bigint` | Minimum output as Wei — enforced on-chain |
| `steps` | `Hex` | Encoded route — passed to `Afi.swap()` |
| `path` | `Address[]` | Token addresses in the route |
| `hops` | `Hop[]` | Per-hop breakdown |
| `slippage` | `number` | Applied slippage |
| `feeBps` | `number` | Protocol fee (basis points) |
| `tokenInPrice` | `string` | Exchange rate of tokenIn in tokenOut |
| `tokenOutPrice` | `string` | Exchange rate of tokenOut in tokenIn |
| `tokenInBasePrice?` | `string` | Price of tokenIn in priceBase asset |
| `tokenOutBasePrice?` | `string` | Price of tokenOut in priceBase asset |

---

### `getFeeBps()` / `GetFeeBps(ctx)`

Reads `feeBps` from the AFI contract. Already included in every `Quote`.

---

### `setApiUrl(url)` / `SetApiURL(url)`

Changes the base URL for API calls (default: `https://rpc.afi.run`). Useful for local development.

```typescript
client.setApiUrl("http://localhost:8080")  // returns this
```

```go
client.SetApiURL("http://localhost:8080")  // returns *Client
```

---

### Signer methods — require `connect(privateKey)` first

#### `connect(privateKey)` / `Connect(privateKey)`

```typescript
client.connect("0xYOUR_PRIVATE_KEY")          // returns this
// or pass at construction: new AfiClient({ rpcUrl, privateKey })
```

```go
err := client.Connect("YOUR_PRIVATE_KEY")
```

#### `executeSwap(quote)` / `ExecuteSwap(ctx, quote)`

Full execution flow from a pre-fetched quote:

```
balance check → approve (exact) → simulate → swap → wait
```

#### `approve(tokenIn, amountWei)` / `Approve(ctx, token, amountWei)`

Returns `PendingTx | null`. `null` means allowance was already sufficient.

#### `simulate(quote, log?)` / `Simulate(ctx, quote, log?)`

Dry-run via `eth_call`. Returns `true` if the swap would succeed.

#### `submitSwap(quote)` / `SubmitSwap(ctx, quote)`

Sends the swap tx. Returns `PendingSwap` with immediate `txHash`.

#### `swap(params)` / `Swap(ctx, ...opts)`

One-call convenience: equivalent to `GetQuote(...)` + `ExecuteSwap(quote)`.

---

## Networks

```typescript
import { NETWORK } from "@afi-run/sdk"
// NETWORK.BASE | NETWORK.BSC | NETWORK.ARBITRUM | NETWORK.ETHEREUM | NETWORK.UNICHAIN
```

```go
// afi.NetworkBase | afi.NetworkBSC | afi.NetworkArbitrum | afi.NetworkEthereum | afi.NetworkUnichain
```

## DEX constants

```typescript
import { DEX } from "@afi-run/sdk"
// DEX.UNI_V3 | DEX.UNI_V4 | DEX.CAKE_V3 | DEX.AERODROME
// DEX.BALANCER | DEX.CURVE128 | DEX.CURVE256 | DEX.FLUID
```

```go
// afi.DexUniV3 | afi.DexUniV4 | afi.DexCakeV3 | afi.DexAerodrome
// afi.DexBalancer | afi.DexCurve128 | afi.DexCurve256 | afi.DexFluid
```

---

## Staged flow

```typescript
const client = new AfiClient({ rpcUrl: "..." })

const quote = await client
  .quote(USDC, WETH, "1000")
  .slippage(0.5)
  .get()

client.connect("0x...")

// 1. Approve
const approval = await client.approve(quote.tokenIn, quote.amountInWei)
if (approval) {
  await approval.wait()
}

// 2. Simulate
const ok = await client.simulate(quote, console.error)
if (!ok) return

// 3. Submit
const pending = await client.submitSwap(quote)
console.log(`Swap tx: ${pending.txHash}`)

// 4. Wait
const result = await pending.wait()
console.log(`Got: ${formatUnits(result.amountOut, 18)} WETH`)
```

```go
client, _ := afi.NewClient(afi.Config{RPCURL: "..."})

quote, _ := client.GetQuote(ctx,
    afi.From(usdc, afi.WETH, "1000"),
    afi.WithSlippage(0.5),
)

client.Connect("YOUR_KEY")

// 1. Approve
approval, _ := client.Approve(ctx, usdc, quote.AmountInWei)
if approval != nil {
    approval.Wait(ctx)
}

// 2. Simulate
ok, _ := client.Simulate(ctx, quote, func(r string) { fmt.Println("Failed:", r) })
if !ok { return }

// 3. Submit
pending, _ := client.SubmitSwap(ctx, quote)
fmt.Println("Swap:", pending.TxHash)

// 4. Wait
result, _ := pending.Wait(ctx)
fmt.Println("Got:", afi.FormatUnits(result.AmountOut, 18), "WETH")
```

---

## Error handling

### Node.js

```typescript
import { NoSignerError, InsufficientBalanceError, SimulationFailedError, QuoteError } from "@afi-run/sdk"

try {
  const result = await client.executeSwap(quote)
} catch (e) {
  if (e instanceof InsufficientBalanceError) {
    console.log("Balance:", e.balance, "Required:", e.required)
  } else if (e instanceof SimulationFailedError) {
    console.log("Would revert:", e.reason)  // no tx was sent
  } else if (e instanceof QuoteError) {
    console.log("No route:", e.message)
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
        case "INSUFFICIENT_BALANCE":
            fmt.Println("Not enough balance")
        case "SIMULATION_FAILED":
            fmt.Println("Would revert:", afiErr.Message) // no tx was sent
        case "QUOTE_FAILED":
            fmt.Println("No route:", afiErr.Message)
        }
    }
}
```

---

## Unit helpers

```typescript
import { parseUnits, formatUnits } from "@afi-run/sdk"

parseUnits("1000", 6)    // 1000000000n
formatUnits(1000000n, 6) // "1000"
```

```go
wei, _ := afi.ParseUnits("1000", 6)   // big.Int
str   := afi.FormatUnits(wei, 6)      // "1000"
```

---

## Constants

| Name | Value |
|---|---|
| AFI contract (Base) | `0xB8cC65321d169D55b93b4402D795701c6B308ce4` |
| WETH (Base) | `0x4200000000000000000000000000000000000006` |
| API base URL | `https://rpc.afi.run` |
| Chain ID (Base) | `8453` |

---

## Examples

### Node.js

| File | What it shows |
|---|---|
| `examples/nodejs/1-list-tokens.ts` | List tokens on Base and BSC |
| `examples/nodejs/2-get-quote.ts` | Builder with all options (priceBase, dexs, Token objects) |
| `examples/nodejs/3-execute-swap.ts` | Quote → review → execute |
| `examples/nodejs/4-full-flow.ts` | One-call `.execute()` with variations |
| `examples/nodejs/5-approve-only.ts` | Staged flow: approve, simulate, submit, wait |

### Go

| Directory | What it shows |
|---|---|
| `examples/go/list-tokens/` | List tokens on Base and BSC |
| `examples/go/get-quote/` | Functional options with all params (priceBase, dexs, Token objects) |
| `examples/go/execute-swap/` | Quote → review → execute |
| `examples/go/full-flow/` | One-call Swap() with variations |
| `examples/go/approve-only/` | Staged flow: approve, simulate, submit, wait |

---

## Build from source

```bash
# Node.js
cd nodejs && npm install && npm run build

# Go
cd go && go build ./...
```
