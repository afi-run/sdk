# AFI SDK

SDK for token swaps on Base via the [AFI Protocol](https://afi.run).

Available in **Node.js (TypeScript)** and **Go**.

---

## Install

```bash
# Go — install directly from GitHub
go get github.com/afi-run/sdk/go

# Node.js — clone and install from local path
git clone https://github.com/afi-run/sdk.git
npm install ./sdk/nodejs
```

> **Node.js note:** npm does not support subdirectory GitHub installs.
> Once the package is published on npm, installation will be simply `npm install @afi-run/sdk`.

---

## Quick start

```typescript
// Node.js
import { AfiClient } from "@afi-run/sdk"

const client = new AfiClient({
  rpcUrl: "https://rpc.ankr.com/base/YOUR_API_KEY",
  privateKey: "0xYOUR_PRIVATE_KEY",
})

// Recommended: quote first, then execute
const quote = await client.getQuote({
  tokenIn:  "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913", // USDC
  tokenOut: "0x4200000000000000000000000000000000000006", // WETH
  amountIn: 1000_000000n,  // 1000 USDC (raw wei, 6 decimals)
  slippage: 0.5,           // 0.5%
})

console.log("Expected out:", quote.amountOutWei)
console.log("Minimum out: ", quote.minOutWei)  // enforced on-chain

const result = await client.executeSwap(quote)
console.log("Tx hash:", result.txHash)
console.log("Got:    ", result.amountOut, "wei WETH")
```

```go
// Go
client, _ := afi.NewClient(afi.Config{
    RPCURL:     "https://rpc.ankr.com/base/YOUR_API_KEY",
    PrivateKey: "YOUR_PRIVATE_KEY",
})
defer client.Close()

amountIn, _ := afi.ParseUnits("1000", 6) // 1000 USDC

quote, _ := client.GetQuote(ctx, afi.SwapParams{
    TokenIn:  common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"),
    TokenOut: afi.WETH,
    AmountIn: amountIn,
    Slippage: 0.5,
})

result, _ := client.ExecuteSwap(ctx, quote)
fmt.Println("Tx:", result.TxHash.Hex())
fmt.Println("Got:", afi.FormatUnits(result.AmountOut, 18), "WETH")
```

---

## API reference

### `getTokens()` — discover available tokens

```typescript
const tokens = await client.getTokens()
// Token[] — tokens active on Base

// Token shape:
// {
//   address:  "0x833589..."
//   symbol:   "USDC"
//   decimals: 6
//   active:   true
// }
```

Hits `GET https://rpc.afi.run/info` and returns the list of tokens
supported on Base. Call this once at startup to let users pick from
valid token addresses. No private key or on-chain interaction needed.

```go
// Go
tokens, err := client.GetTokens(ctx)
for _, t := range tokens {
    fmt.Printf("%s → %s\n", t.Symbol, t.Address.Hex())
}
```

---

### `getQuote(params)` — fetch a price quote

```typescript
const quote = await client.getQuote({
  tokenIn:  "0x...",               // input token address
  tokenOut: "0x...",               // output token address
  amountIn: parseUnits("1000", 6), // 1000 USDC — use parseUnits to avoid raw wei
  slippage: 0.5,                   // percentage, e.g. 0.5 = 0.5%
})
```

**Read-only** — no transaction is sent. Safe to call at any frequency to
show live pricing. Internally it:

1. Reads `decimals()` from the input token contract
2. Reads `feeBps()` from the AFI contract (live — fee can change)
3. Calls `POST https://rpc.afi.run/quoter` with your RPC URL

**Quote fields:**

| Field | Type | Description |
|---|---|---|
| `amountInWei` | `bigint` | Exact amount to approve and send |
| `amountOutWei` | `bigint` | Estimated output (informational) |
| `minOutWei` | `bigint` | Minimum output after slippage — enforced on-chain |
| `steps` | `Hex` | Encoded route passed to `Afi.swap()` — do not modify |
| `path` | `Address[]` | Token addresses in the route |
| `slippage` | `number` | Applied slippage percentage |
| `feeBps` | `number` | Protocol fee at the time of quote (basis points) |

**Important:** `minOutWei` is never 0. The SDK rejects quotes with zero minimum output.

---

### `executeSwap(quote)` — execute a pre-fetched quote

```typescript
const result = await client.executeSwap(quote)
```

Takes a `Quote` returned by `getQuote()` and runs the full execution flow:

```
1. assertBalance     — verifies tokenIn balance ≥ amountInWei
2. approve           — approves exactly amountInWei to AFI contract
                       (skipped if existing allowance is already sufficient)
3. simulate          — runs eth_call before sending — throws if swap would revert
4. swap              — sends the transaction with 1.2× gas estimate
5. parse event       — waits for receipt, reads actual amounts from SwapExecuted
```

**Why simulate first?** If the swap would revert (e.g. price moved past
`minOut`), `SimulationFailedError` is thrown before any gas is spent.

**Result fields:**

| Field | Type | Description |
|---|---|---|
| `txHash` | `Hex` | Transaction hash |
| `blockNumber` | `bigint` | Block where the swap was confirmed |
| `amountIn` | `bigint` | Actual input from on-chain `SwapExecuted` event |
| `amountOut` | `bigint` | Actual output from on-chain `SwapExecuted` event |
| `gasUsed` | `bigint` | Gas consumed by the transaction |

---

### `swap(params)` — convenience: quote + execute in one call

```typescript
const result = await client.swap({
  tokenIn:  "0x...",
  tokenOut: "0x...",
  amountIn: 500_000000n,
  slippage: 1.0,
})
```

Equivalent to `const q = await getQuote(params); return executeSwap(q)`.

Use this for bots and scripts. For user-facing apps, prefer the
`getQuote` → show pricing → `executeSwap` pattern so users can review
before confirming.

---

### `approve(tokenIn, amountWei)` — approve only

```typescript
const txHash = await client.approve(tokenIn, quote.amountInWei)
// Returns null if allowance was already sufficient (no tx sent)
```

Approves exactly `amountWei` — no more — to the AFI contract.

`executeSwap()` calls this automatically. Use it directly only if your
app needs to show the approve and swap as two separate wallet prompts.

**Safety details:**
- Checks existing allowance on-chain first — skips if already sufficient
- Resets to 0 before re-approving for USDT-style tokens that require it
- Re-verifies allowance on-chain after the approval tx confirms

---

### `getFeeBps()` — read current protocol fee

```typescript
const feeBps = await client.getFeeBps()
// e.g. 35 → 0.35%
```

Reads `feeBps` directly from the AFI contract. The fee can change and
is already included in every `Quote` object returned by `getQuote()`.

---

### `parseUnits(amount, decimals)` / `formatUnits(amount, decimals)` — unit helpers

```typescript
import { parseUnits, formatUnits } from "@afi-run/sdk"

// Human-readable → raw wei (bigint) — use as amountIn
parseUnits("1000", 6)          // 1000_000000n  (1000 USDC)
parseUnits("1.5", 6)           // 1_500000n
parseUnits("0.5", 18)          // 500000000000000000n  (0.5 WETH)

// Raw wei (bigint) → human-readable — use to display amounts
formatUnits(1000_000000n, 6)   // "1000"
formatUnits(1_500000n, 6)      // "1.5"
formatUnits(500000000000000000n, 18) // "0.5"
```

```go
// Go
wei, err := afi.ParseUnits("1000", 6)   // big.Int 1000_000000
str := afi.FormatUnits(wei, 6)          // "1000"
```

These helpers let you work with the amounts your users type instead of raw wei:

```typescript
// Without helpers
const quote = await client.getQuote({
  amountIn: 1000_000000n,  // must know USDC has 6 decimals
  ...
})

// With helpers
const quote = await client.getQuote({
  amountIn: parseUnits("1000", 6),  // readable
  ...
})
```

---

## Approval: why always exact?

The SDK approves exactly the amount from the quote — never more. This means:

- Even if the AFI contract were compromised, an attacker could only spend the
  amount you were already going to spend in that specific swap
- If you swap frequently, you'll send one approval transaction per swap
- `executeSwap()` skips the approval if your existing allowance is already sufficient

---

## Security guarantees

| Risk | How the SDK handles it |
|---|---|
| Slippage bypass | `minOut` always comes from the quoter API — never set to 0 |
| Approve too much | Always approves exactly `amountInRaw` from the quote |
| USDT-style tokens | Resets allowance to 0 before re-approving if needed |
| Tx reverts | `eth_call` simulation runs before every swap — fails fast |
| Race condition (allow vs swap) | Allowance re-verified on-chain after approval confirms |
| Gas underestimate | Gas estimated on-chain then multiplied by 1.2 |
| Native ETH passed | Not supported — use WETH: `0x4200000000000000000000000000000000000006` |

---

## Error handling

### Node.js

```typescript
import {
  InsufficientBalanceError,
  SimulationFailedError,
  QuoteError,
  ApprovalError,
  SwapRevertedError,
} from "@afi-run/sdk"

try {
  const result = await client.executeSwap(quote)
} catch (e) {
  if (e instanceof InsufficientBalanceError) {
    // User doesn't have enough tokenIn
    console.log("Balance:", e.balance)   // bigint, raw wei
    console.log("Required:", e.required) // bigint, raw wei
    console.log("Token:", e.token)       // address string

  } else if (e instanceof SimulationFailedError) {
    // Swap would revert — no transaction was sent, no gas spent
    console.log("Reason:", e.reason)       // decoded revert string
    console.log("Data:", e.revertData)     // raw revert bytes (optional)

  } else if (e instanceof QuoteError) {
    // Quoter API returned an error (e.g. no route found)
    console.log(e.message)

  } else if (e instanceof ApprovalError) {
    // Token approval transaction failed
    console.log(e.message)

  } else if (e instanceof SwapRevertedError) {
    // Swap transaction reverted on-chain
    console.log("Reason:", e.reason)
  }
}
```

### Go

```go
import "errors"

result, err := client.ExecuteSwap(ctx, quote)
if err != nil {
    var afiErr *afi.AfiError
    if errors.As(err, &afiErr) {
        switch afiErr.Code {
        case "INSUFFICIENT_BALANCE":
            fmt.Println("Not enough balance")
        case "SIMULATION_FAILED":
            // No tx was sent
            fmt.Println("Would revert:", afiErr.Message)
        case "QUOTE_FAILED":
            fmt.Println("No route found:", afiErr.Message)
        case "APPROVAL_FAILED":
            fmt.Println("Approval failed:", afiErr.Message)
        case "SWAP_REVERTED":
            fmt.Println("Swap reverted:", afiErr.Message)
        }
        return
    }
    log.Fatal(err) // unexpected error
}
```

---

## Constants

| Name | Value |
|---|---|
| AFI contract (Base) | `0xB8cC65321d169D55b93b4402D795701c6B308ce4` |
| WETH (Base) | `0x4200000000000000000000000000000000000006` |
| Quoter API | `https://rpc.afi.run/quoter` |
| Info API | `https://rpc.afi.run/info` |
| Chain ID | `8453` |

```typescript
import { AFI_ADDRESS, WETH } from "@afi-run/sdk"
```

```go
afi.AfiAddress // common.Address
afi.WETH       // common.Address
```

---

## Examples

### Node.js

| File | What it shows |
|---|---|
| `examples/nodejs/1-list-tokens.ts` | List all supported tokens |
| `examples/nodejs/2-get-quote.ts` | Fetch and inspect a quote |
| `examples/nodejs/3-execute-swap.ts` | Quote → review → execute (recommended) |
| `examples/nodejs/4-full-flow.ts` | One-call convenience swap |
| `examples/nodejs/5-approve-only.ts` | Separate approve and swap steps |

```bash
cd nodejs
npm install
npx ts-node ../examples/nodejs/1-list-tokens.ts
```

### Go

| Directory | What it shows |
|---|---|
| `examples/go/list-tokens/` | List all supported tokens |
| `examples/go/get-quote/` | Fetch and inspect a quote |
| `examples/go/execute-swap/` | Quote → review → execute (recommended) |
| `examples/go/full-flow/` | One-call convenience swap |
| `examples/go/approve-only/` | Separate approve and swap steps |

```bash
cd examples/go
go mod tidy
go run ./list-tokens
go run ./get-quote
go run ./execute-swap
```

---

## Build from source

```bash
# Node.js
cd nodejs
npm install
npm run build        # outputs to dist/
npm run typecheck    # type check only

# Go
cd go
go mod tidy
go build ./...
```
